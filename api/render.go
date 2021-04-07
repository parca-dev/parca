package api

import (
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/google/pprof/profile"
)

type ProfileResponseRenderer struct {
	logger   log.Logger
	profile  *profile.Profile
	warnings []error
	req      *http.Request
}

func NewProfileResponseRenderer(
	logger log.Logger,
	profile *profile.Profile,
	warnings []error,
	req *http.Request,
) *ProfileResponseRenderer {
	return &ProfileResponseRenderer{
		logger:   logger,
		profile:  profile,
		warnings: warnings,
		req:      req,
	}
}

func (r *ProfileResponseRenderer) Render(w http.ResponseWriter) error {
	switch r.req.URL.Query().Get("report") {
	case "meta":
		meta, err := GenerateMetaReport(r.profile)
		if err != nil {
			return err
		}

		return NewSuccessResponse(meta, r.warnings).Render(w)
	case "top":
		top, err := generateTopReport(r.profile, r.req.URL.Query().Get("sample_index"))
		if err != nil {
			return err
		}

		return NewSuccessResponse(top, r.warnings).Render(w)
	case "flamegraph":
		fg, err := generateFlamegraphReport(r.profile, r.req.URL.Query().Get("sample_index"))
		if err != nil {
			return err
		}

		return NewSuccessResponse(fg, r.warnings).Render(w)
	case "proto":
		return NewProtoRenderer(r.profile).Render(w)
	case "svg":
		return NewSVGRenderer(
			r.logger,
			r.profile,
			r.req.URL.Query().Get("sample_index"),
		).Render(w)
	default:
		return NewSVGRenderer(
			r.logger,
			r.profile,
			r.req.URL.Query().Get("sample_index"),
		).Render(w)
	}
}

type ValueType struct {
	Type string `json:"type,omitempty"`
}

type MetaReport struct {
	SampleTypes       []ValueType `json:"sampleTypes"`
	DefaultSampleType string      `json:"defaultSampleType"`
}

func GenerateMetaReport(profile *profile.Profile) (*MetaReport, error) {
	index, err := profile.SampleIndexByName("")
	if err != nil {
		return nil, err
	}

	res := &MetaReport{
		SampleTypes:       []ValueType{},
		DefaultSampleType: profile.SampleType[index].Type,
	}
	for _, t := range profile.SampleType {
		res.SampleTypes = append(res.SampleTypes, ValueType{t.Type})
	}

	return res, nil
}

type ProtoRenderer struct {
	profile *profile.Profile
}

func NewProtoRenderer(profile *profile.Profile) *ProtoRenderer {
	return &ProtoRenderer{profile: profile}
}

func (r *ProtoRenderer) Render(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/vnd.google.protobuf+gzip")
	w.Header().Set("Content-Disposition", "attachment;filename=profile.pb.gz")
	err := r.profile.Write(w)
	if err != nil {
		return err
	}
	return nil
}
