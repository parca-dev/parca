// Copyright 2024-2026 The Parca Authors
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

package demangle

import (
	"testing"

	"github.com/ianlancetaylor/demangle"
	"github.com/stretchr/testify/require"
)

func TestDemangleRemoveRustTypeParameters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		mangledName string
		expected    string
	}{
		{
			name:        "first",
			mangledName: "_RNvXs0_NtNtNtNtCscFZsgO8y1ob_6timely8dataflow9operators7generic11builder_rawINtB5_12OperatorCoreNtNtCs3rN31ftiYwT_7mz_repr9timestamp9TimestampNCINvMNtB7_10builder_rcINtB2m_15OperatorBuilderINtNtNtBb_6scopes5child5ChildIB32_IB32_IB32_INtNtBd_6worker6WorkerNtNtNtCsaCQYdX5e1rn_20timely_communication9allocator7generic7GenericEB1w_EB1w_EB1w_EB1w_EE16build_rescheduleNCINvB2l_5buildNCINvXNtB7_8operatorINtNtBb_6stream10StreamCoreB31_INtNtCs9Pxm3sdlyVG_5alloc3vec3VecTTTNtNtB1A_3row3RowyEIB6U_B7t_EEB1w_xEEEINtB6a_8OperatorB31_B6T_E14unary_frontierIB6U_INtNtB6Y_2rc2RcINtNtNtNtCs3pxblAp50HC_21differential_dataflow5trace15implementations3ord11OrdValBatchB7s_B7L_B1w_xjINtNtCs2MRLV2eX2T0_16timely_container11columnation11TimelyStackB7s_EIBaG_B7L_EEEENCINvXs1_NtNtNtB9c_9operators7arrange11arrangementINtNtB9c_10collection10CollectionB31_B7r_xEINtBc7_7ArrangeB31_B7s_B7L_xE12arrange_coreINtNtNtBb_8channels4pact12ExchangeCoreB6T_B7q_NCINvBc3_13arrange_namedINtNtB98_12spine_fueled5SpineB8O_EE0EBfi_E0NCNCBc0_00Bea_E0NCNCB66_00E0NCNCB5R_00E0ENtNtBd_10scheduling8Schedule8scheduleCs4ogIrgXwtlZ_8clusterd",
			expected:    "<timely::dataflow::operators::generic::builder_raw::OperatorCore<> as timely::scheduling::Schedule>::schedule",
		},
		{
			name:        "second",
			mangledName: "_RNCINvMNtNtNtNtCs5myfTy8mnaF_6timely8dataflow9operators7generic10builder_rcINtB5_15OperatorBuilderINtNtNtBb_6scopes5child5ChildIB1z_INtNtBd_6worker6WorkerNtNtNtCsbo5udLplCaV_20timely_communication9allocator7generic7GenericENtNtCslnPiKci8RgF_7mz_repr9timestamp9TimestampEB3z_EE16build_rescheduleNCINvB4_5buildNCINvXNtB7_8operatorINtNtBb_6stream10StreamCoreB1y_INtNtCsfohDMHpnFpV_5alloc3vec3VecTTNtNtB3D_3row3RowB6k_EB3z_xEEEINtB52_8OperatorB1y_B5L_E14unary_frontierIB5M_INtNtB5Q_2rc2RcINtNtNtNtCsaEm0OTy3LfN_21differential_dataflow5trace15implementations3ord11OrdValBatchB6k_B6k_B3z_xjINtNtCsicJTUUNBAMQ_16timely_container11columnation11TimelyStackB6k_EB9o_EEENCINvXs1_NtNtNtB7V_9operators7arrange11arrangementINtNtB7V_10collection10CollectionB1y_B6j_xEINtBaK_7ArrangeB1y_B6k_B6k_xE12arrange_coreINtNtNtBb_8channels4pact12ExchangeCoreB5L_B6i_NCINvBaG_13arrange_namedINtNtB7R_12spine_fueled5SpineB7x_EE0EBdV_E0NCNCBaD_00BcN_E0NCNCB4Y_00E0NCNCB4K_00E0Cse28fqe15ASj_8clusterd",
			expected:    "<timely::dataflow::operators::generic::builder_rc::OperatorBuilder<>>::build_reschedule::<>::{closure#0}",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			d := newDemangler([]demangle.Option{demangle.NoParams, demangle.NoTemplateParams})
			demangledName := d.Demangle([]byte(c.mangledName))

			require.Equal(t, c.expected, demangledName)
		})
	}
}

func BenchmarkDemangleRemoveRustTypeParameters(b *testing.B) {
	mangledName := "_RNvXs0_NtNtNtNtCscFZsgO8y1ob_6timely8dataflow9operators7generic11builder_rawINtB5_12OperatorCoreNtNtCs3rN31ftiYwT_7mz_repr9timestamp9TimestampNCINvMNtB7_10builder_rcINtB2m_15OperatorBuilderINtNtNtBb_6scopes5child5ChildIB32_IB32_IB32_INtNtBd_6worker6WorkerNtNtNtCsaCQYdX5e1rn_20timely_communication9allocator7generic7GenericEB1w_EB1w_EB1w_EB1w_EE16build_rescheduleNCINvB2l_5buildNCINvXNtB7_8operatorINtNtBb_6stream10StreamCoreB31_INtNtCs9Pxm3sdlyVG_5alloc3vec3VecTTTNtNtB1A_3row3RowyEIB6U_B7t_EEB1w_xEEEINtB6a_8OperatorB31_B6T_E14unary_frontierIB6U_INtNtB6Y_2rc2RcINtNtNtNtCs3pxblAp50HC_21differential_dataflow5trace15implementations3ord11OrdValBatchB7s_B7L_B1w_xjINtNtCs2MRLV2eX2T0_16timely_container11columnation11TimelyStackB7s_EIBaG_B7L_EEEENCINvXs1_NtNtNtB9c_9operators7arrange11arrangementINtNtB9c_10collection10CollectionB31_B7r_xEINtBc7_7ArrangeB31_B7s_B7L_xE12arrange_coreINtNtNtBb_8channels4pact12ExchangeCoreB6T_B7q_NCINvBc3_13arrange_namedINtNtB98_12spine_fueled5SpineB8O_EE0EBfi_E0NCNCBc0_00Bea_E0NCNCB66_00E0NCNCB5R_00E0ENtNtBd_10scheduling8Schedule8scheduleCs4ogIrgXwtlZ_8clusterd"

	d := newDemangler([]demangle.Option{demangle.NoParams, demangle.NoTemplateParams})

	var dontOptimize string
	for i := 0; i < b.N; i++ {
		dontOptimize = d.Demangle([]byte(mangledName))
	}
	_ = dontOptimize
}
