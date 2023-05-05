// Copyright 2022-2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metastore

import (
	"testing"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

func TestMakeLocationID(t *testing.T) {
	tests := map[string]struct {
		loc  *pb.Location
		want string
	}{
		"one line": {
			loc: &pb.Location{
				Address:   9424419,
				MappingId: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=",
				Lines: []*pb.Line{
					{
						FunctionId: "dFRfEsG-uRAep6DtB_p4adjSQiRipfgf0LTZ3B05k74=/auEFFJYKPIUbrC2nw-kBi9ePl80B2bwY6mCyCpUeC78=",
						Line:       206,
					},
				},
			},
			want: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/kBL4fDvKSLKe5R_x08kcKOji7UXafFAYjquOeVeZsrA=",
		},
		"two lines": {
			loc: &pb.Location{
				Address:   9287454,
				MappingId: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=",
				Lines: []*pb.Line{
					{
						FunctionId: "M0phtCvUI0mtpx5-BIqngA332lj6UjfSV64urDi6C1U=/sMz0kkKiNbEqtL2vipylGI9f2Lc-M1ExWCZWu9KKo5I=",
						Line:       88,
					},
					{
						FunctionId: "APOSsLfLXhP_xNKKKwdpqtMv1ROA5u2xTbWCHYxFIVM=/vSrEl_sMYM6DXy8DmzcBC2yxXNr_Duv0hrrHZLeVws0=",
						Line:       41,
					},
					{
						FunctionId: "IXXv_eDgJGGpb7ikH21IOdbpfBzTPuRWklMHyheBez4=/joa-isXk0b6wK7TcTHqFVXm1Z5uG17Wpd44wuo8LYqA=",
						Line:       201,
					},
				},
			},
			want: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/q2qKTcQ2-RK6Hz06jcBWjT_mve0TYLGYXlFLCUzFdXs=",
		},
	}

	km := NewKeyMaker()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := km.MakeLocationID(tc.loc)
			if tc.want != got {
				t.Errorf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkMakeLocationID(b *testing.B) {
	km := NewKeyMaker()
	loc := &pb.Location{
		Address:   9287454,
		MappingId: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=",
		Lines: []*pb.Line{
			{
				FunctionId: "M0phtCvUI0mtpx5-BIqngA332lj6UjfSV64urDi6C1U=/sMz0kkKiNbEqtL2vipylGI9f2Lc-M1ExWCZWu9KKo5I=",
				Line:       88,
			},
			{
				FunctionId: "APOSsLfLXhP_xNKKKwdpqtMv1ROA5u2xTbWCHYxFIVM=/vSrEl_sMYM6DXy8DmzcBC2yxXNr_Duv0hrrHZLeVws0=",
				Line:       41,
			},
			{
				FunctionId: "IXXv_eDgJGGpb7ikH21IOdbpfBzTPuRWklMHyheBez4=/joa-isXk0b6wK7TcTHqFVXm1Z5uG17Wpd44wuo8LYqA=",
				Line:       201,
			},
		},
	}
	want := "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/q2qKTcQ2-RK6Hz06jcBWjT_mve0TYLGYXlFLCUzFdXs="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := km.MakeLocationID(loc)
		if want != got {
			b.Errorf("expected %q got %q", want, got)
		}
	}
}

func TestMakeFunctionID(t *testing.T) {
	tests := map[string]struct {
		f    *pb.Function
		want string
	}{
		"k8s": {
			f: &pb.Function{
				Name:       "k8s.io/apimachinery/pkg/util/net.CloneHeader",
				SystemName: "k8s.io/apimachinery/pkg/util/net.CloneHeader",
				Filename:   "/home/runner/go/pkg/mod/k8s.io/apimachinery@v0.19.2/pkg/util/net/http.go",
			},
			want: "LyobRlcs0hfG2gj9yNHcjcCQoZsgdyH_1PgJoMRu1qI=/BGYf55XWQ8LntSPMGLsYojB-CvbvjzfoW5sUhYuy1Pw=",
		},
		"prom": {
			f: &pb.Function{
				Name:       "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1.newAlertmanagerConfigs",
				SystemName: "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1.newAlertmanagerConfigs",
				Filename:   "/home/runner/work/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1/alertmanagerconfig.go",
			},
			want: "aHfN-tXDeVXU8Dcm2cYE8JFXiUJaseYC8kkI8OfSr3w=/qNCGQxzcAbzz1sFpEnk_f3sqxYy3UlHqmf57-xiRxgo=",
		},
	}

	km := NewKeyMaker()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := km.MakeFunctionID(tc.f)
			if tc.want != got {
				t.Errorf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkMakeFunctionID(b *testing.B) {
	km := NewKeyMaker()
	f := &pb.Function{
		Name:       "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1.newAlertmanagerConfigs",
		SystemName: "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1.newAlertmanagerConfigs",
		Filename:   "/home/runner/work/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1alpha1/alertmanagerconfig.go",
	}
	want := "aHfN-tXDeVXU8Dcm2cYE8JFXiUJaseYC8kkI8OfSr3w=/qNCGQxzcAbzz1sFpEnk_f3sqxYy3UlHqmf57-xiRxgo="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := km.MakeFunctionID(f)
		if want != got {
			b.Errorf("expected %q got %q", want, got)
		}
	}
}
