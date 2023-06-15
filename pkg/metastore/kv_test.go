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

func TestMakeMappingID(t *testing.T) {
	tests := map[string]struct {
		m    *pb.Mapping
		want string
	}{
		"buildID": {
			m: &pb.Mapping{
				Start:   4194304,
				Limit:   4603904,
				BuildId: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
			},
			want: "NV3TGa0pQ3Xyt-oqzJOF7HklQRs8uJXtO-koTl8ySow=",
		},
		"file has_functions": {
			m: &pb.Mapping{
				Start:        4194304,
				Limit:        4898816,
				File:         "/vagrant/parca/pkg/parca/testdata/pgotest",
				HasFunctions: true,
			},
			want: "jzVUbRN-7tBCz4LbEk7h4AEuH5BbsjOJgh41zsoMw24=",
		},
		"file": {
			m: &pb.Mapping{
				Start: 140729113411584,
				Limit: 140729113419776,
				File:  "[vdso]",
			},
			want: "EhTELjgEXjN888CueYVJRkvMxNFAAPEYmrsINwviMtA=",
		},
	}

	km := NewKeyMaker()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := km.MakeMappingID(tc.m)
			if tc.want != got {
				t.Errorf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkMakeMappingID(b *testing.B) {
	km := NewKeyMaker()
	m := &pb.Mapping{
		Start:   4194304,
		Limit:   4603904,
		BuildId: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}
	want := "NV3TGa0pQ3Xyt-oqzJOF7HklQRs8uJXtO-koTl8ySow="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := km.MakeMappingID(m)
		if want != got {
			b.Errorf("expected %q got %q", want, got)
		}
	}
}

func TestMakeStacktraceID(t *testing.T) {
	tests := map[string]struct {
		s    *pb.Stacktrace
		want string
	}{
		"no location IDs": {
			s: &pb.Stacktrace{
				LocationIds: nil,
			},
			want: "empty-stacktrace",
		},
		"location IDs": {
			s: &pb.Stacktrace{
				LocationIds: []string{
					"2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/6-T7FujFgmEW5lxsVl5_Vyns3HwT4GDRzyb1XR9ywNo=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/QYu_0KTF2Gt1K5G2QG1aB_H_yfkRa6E4U2_uzQ-Xuv8=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/vYMTDe0aPeGnhicxwpMri1VlFRZ20XRnlZ_8Htr6xdY=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/oNOzXHB9JA3nvmJv8tupxLvLTSt18R6KaMoCkCwVU7I=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/oNOzXHB9JA3nvmJv8tupxLvLTSt18R6KaMoCkCwVU7I=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/5tY465_RJPfYD1SpeKedQ0IPSmL-iHawLd2ST1R4_Vk=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/ALOMngTNZMGufgSVyrWYyHwOalaKORZAG-jd2i4EQSc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/lrwPh7-60oemzHnE3POSFhgYl7LScSYnZ4TepL3KsUU=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/vTOjUx_ljAU2FZqyrByg_OY2cgo0X7AUSYilj7ZByd4=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/lNuBC6PJiVsGu8cVU1FIwdnwppcawOvY2PFDlcAR45Q=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/xBWYu4TkB3nQvGcGLB0cGmGLmhwU19pI5M9e8qSFGYU=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/jX38ZG3Vief3Kmfl0MBAlylr-cess1OUfiMWJZsfvjA=",
				},
			},
			want: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/jX38ZG3Vief3Kmfl0MBAlylr-cess1OUfiMWJZsfvjA=/mUeut4y9HHvRshC19OuNRSi19_88qXBoekXsC6w2K00=",
		},
	}

	km := NewKeyMaker()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := km.MakeStacktraceID(tc.s)
			if tc.want != got {
				t.Errorf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkMakeStacktraceID(b *testing.B) {
	km := NewKeyMaker()
	s := &pb.Stacktrace{
		LocationIds: []string{
			"2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/6-T7FujFgmEW5lxsVl5_Vyns3HwT4GDRzyb1XR9ywNo=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/QYu_0KTF2Gt1K5G2QG1aB_H_yfkRa6E4U2_uzQ-Xuv8=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/vYMTDe0aPeGnhicxwpMri1VlFRZ20XRnlZ_8Htr6xdY=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/oNOzXHB9JA3nvmJv8tupxLvLTSt18R6KaMoCkCwVU7I=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/oNOzXHB9JA3nvmJv8tupxLvLTSt18R6KaMoCkCwVU7I=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/d3UCFTQGH24y755FgDF0NDFtdc3txunOkzvWcXQstzc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/cNytybYd5Pr3-XsIG-n2a7UrW3gfyHj-ivHc1c6Mhqc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/SjCsGLa0587ijwzDLRe6MEb3ZlR-U-1Yl31e_KX8-TM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/0SPf0RRit-KuVet_G-gUVOn5HgD4O0FLckf68nOHD4c=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/hpdra4vpDkfwD1jbx3FBtLe15aUoJLSLlkHjMRPsfWM=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/5tY465_RJPfYD1SpeKedQ0IPSmL-iHawLd2ST1R4_Vk=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/ALOMngTNZMGufgSVyrWYyHwOalaKORZAG-jd2i4EQSc=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/lrwPh7-60oemzHnE3POSFhgYl7LScSYnZ4TepL3KsUU=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/vTOjUx_ljAU2FZqyrByg_OY2cgo0X7AUSYilj7ZByd4=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/lNuBC6PJiVsGu8cVU1FIwdnwppcawOvY2PFDlcAR45Q=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/xBWYu4TkB3nQvGcGLB0cGmGLmhwU19pI5M9e8qSFGYU=", "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/jX38ZG3Vief3Kmfl0MBAlylr-cess1OUfiMWJZsfvjA=",
		},
	}
	want := "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/jX38ZG3Vief3Kmfl0MBAlylr-cess1OUfiMWJZsfvjA=/mUeut4y9HHvRshC19OuNRSi19_88qXBoekXsC6w2K00="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := km.MakeStacktraceID(s)
		if want != got {
			b.Errorf("expected %q got %q", want, got)
		}
	}
}
