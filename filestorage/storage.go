package filestorage

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
)

type FileStorage struct {
	dir string
	sync.RWMutex
	retention time.Duration
	logger    log.Logger
}

type ShardMetadata struct {
	Files []FileMedatada
}

type FileMedatada struct {
	Name   string
	Time   time.Time
	Labels labels.Labels
}

func NewFileStorage(dir string, retention time.Duration, logger log.Logger) *FileStorage {
	fs := &FileStorage{
		dir:       dir,
		retention: retention,
		logger:    logger,
	}

	go func() {
		for range time.Tick(time.Minute * 10) {
			if err := fs.DeleteOld(); err != nil {
				level.Error(logger).Log("msg", "Error deleting old files", "err", err)
			}
		}
	}()

	return fs
}

func (fs *FileStorage) LoadMetadata(dir string) (*ShardMetadata, error) {
	var meta ShardMetadata
	p := path.Join(dir, "metadata.json")

	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &ShardMetadata{}, nil
		}
		return nil, errors.Wrap(err, "couldn't open metadata file")
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&meta)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't decode metadata file")
	}

	return &meta, nil
}

func (fs *FileStorage) SaveMetadata(dir string, meta *ShardMetadata) error {
	p := path.Join(dir, "metadata.json")

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return errors.Wrap(err, "couldn't make data directory")
	}

	f, err := os.Create(p)
	if err != nil {
		return errors.Wrap(err, "couldn't create metadata file")
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(meta)
	if err != nil {
		return errors.Wrap(err, "couldn't encode metadata file")
	}

	return nil
}

func (fs *FileStorage) Create(l labels.Labels, t time.Time) (io.WriteCloser, error) {
	fs.Lock()
	defer fs.Unlock()
	for i := 0; i < len(l); {
		if len(l[i].Name) == 0 {
			l[i] = l[len(l)-1]
			l = l[:len(l)-1]
		} else {
			i++
		}
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].Name < l[j].Name
	})

	directory := fmt.Sprintf("%s/%v", fs.dir, t.Unix()/60) // Minutes since epoch start
	meta, err := fs.LoadMetadata(directory)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load shard metadata")
	}
	meta.Files = append(meta.Files, FileMedatada{
		Name:   fmt.Sprint(len(meta.Files)),
		Time:   t,
		Labels: l,
	})
	err = fs.SaveMetadata(directory, meta)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't save shard metadata")
	}
	fileName := path.Join(directory, meta.Files[len(meta.Files)-1].Name)

	f, err := os.Create(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open file")
	}
	return f, nil
}

func (fs *FileStorage) ListSeries(from, to time.Time, matchers ...*labels.Matcher) (map[string][]FileMedatada, error) {
	fs.RLock()
	defer fs.RUnlock()

	directories, err := ioutil.ReadDir(fs.dir)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read directory")
	}

	seriesSet := make(map[string][]FileMedatada)

	for _, dir := range directories {
		pointInTime, err := strconv.ParseInt(dir.Name(), 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse directory name as minute")
		}
		if from.Unix() >= pointInTime*60 || to.Unix() <= pointInTime*60 { // Scale from minutes since epoch start
			continue
		}
		files, err := fs.MatchFiles(path.Join(fs.dir, dir.Name()), matchers...)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't match files in directory %v", dir.Name())
		}

		for _, file := range files {
			seriesSet[file.Labels.String()] = append(seriesSet[file.Labels.String()], file)
		}
	}

	return seriesSet, nil
}

func (fs *FileStorage) MatchFiles(dir string, matchers ...*labels.Matcher) ([]FileMedatada, error) {
	out := make([]FileMedatada, 0)

	meta, err := fs.LoadMetadata(dir)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load shard metadata")
	}
	for _, file := range meta.Files {
		allMatch := true
	matcherLoop:
		for _, matcher := range matchers {
			for _, label := range file.Labels {
				if matcher.Name == label.Name && matcher.Matches(label.Value) {
					continue matcherLoop
				}
			}
			allMatch = false
			break
		}
		if allMatch {
			out = append(out, file)
		}
	}

	return out, nil
}

func (fs *FileStorage) GetFile(t time.Time, matchers ...*labels.Matcher) (io.ReadCloser, error) {
	fs.RLock()
	defer fs.RUnlock()

	meta, err := fs.LoadMetadata(path.Join(fs.dir, fmt.Sprint(t.Unix()/60))) // Minutes since epoch start
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load shard metadata")
	}
	for _, file := range meta.Files {
		if file.Time.Equal(t) {
			continue
		}
		allMatch := true
	matcherLoop:
		for _, matcher := range matchers {
			for _, label := range file.Labels {
				if matcher.Name == label.Name && matcher.Matches(label.Value) {
					continue matcherLoop
				}
			}
			allMatch = false
			break
		}
		if allMatch {
			f, err := os.Open(path.Join(fs.dir, fmt.Sprint(t.Unix()/(60)), file.Name)) // Minutes since epoch start
			if err != nil {
				return nil, errors.Wrap(err, "couldn't open pprof file")
			}
			return f, nil
		}
	}

	return nil, errors.Errorf("file not found")
}

func (fs *FileStorage) DeleteOld() error {
	fs.Lock()
	defer fs.Unlock()

	directories, err := ioutil.ReadDir(fs.dir)
	if err != nil {
		return errors.Wrap(err, "couldn't read directory")
	}

	for _, dir := range directories {
		pointInTime, err := strconv.ParseInt(dir.Name(), 10, 64)
		if err != nil {
			return errors.Wrap(err, "couldn't parse directory name as minute")
		}
		if pointInTime*60 <= time.Now().Add(-1*fs.retention).Unix() { // Scale from minutes since epoch start
			err := os.RemoveAll(path.Join(fs.dir, dir.Name()))
			if err != nil {
				return errors.Wrapf(err, "couldn't delete old directory: %v", path.Join(fs.dir, dir.Name()))
			}
		}
	}

	return nil
}
