// Copyright 2022 The Parca Authors
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

// Descriptions for GPU PC-sampling frames shown in the flamegraph tooltip.
//
// Two name spaces appear as leaf frames in CUDA PC-sampling profiles:
//
//   1. SASS mnemonics — uppercase base opcodes (modifiers like `.E`, `.128`,
//      `.FTZ` stripped). Source: NVIDIA PTX ISA + opcode tables in
//      https://github.com/gnurizen/sass-table .
//
//   2. CUPTI / Nsight Compute warp PC-sampling reasons — lower-case metric
//      names of the shape `smsp__pcsamp_warps_issue_<state>`. Source:
//      Nsight Compute "Warp Scheduler States / Stall Reasons" reference
//      (https://docs.nvidia.com/nsight-compute/ProfilingGuide/index.html).
//
// Both lookups are exact-match — there is no risk of colliding with user
// function names.
// To add a new entry, just append a row.

export const SASS_INSTRUCTION_DESCRIPTIONS: Record<string, string> = {
  // --- Integer arithmetic ---
  IADD: 'Integer add.',
  IADD3: 'Three-operand integer add.',
  ISUB: 'Integer subtract.',
  IMUL: 'Integer multiply.',
  IMAD: 'Integer multiply-add.',
  IMADSP: 'Integer multiply-add with subword packing.',
  IMNMX: 'Integer min / max.',
  IABS: 'Integer absolute value.',
  ISCADD: 'Scale (left-shift) then add — fast scaled add.',
  IDP: 'Integer dot product.',
  VABSDIFF: 'Video instruction: absolute difference of integers.',
  VABSDIFF4: 'Packed 8-bit absolute difference.',

  // --- Integer compare / set ---
  ISETP: 'Integer compare-and-set predicate.',
  ISET: 'Integer compare-and-set register.',
  ICMP: 'Integer compare-and-select.',

  // --- Bit ops / shift ---
  LOP: 'Bitwise logical operation (AND / OR / XOR).',
  LOP3: 'Three-input bitwise logic via 8-bit lookup table (arbitrary logic).',
  PLOP3: 'Three-input predicate logic via 8-bit lookup table.',
  SHL: 'Shift left.',
  SHR: 'Shift right.',
  SHF: 'Funnel shift (combined left/right shift across two registers).',
  BFE: 'Bit-field extract.',
  BFI: 'Bit-field insert.',
  POPC: 'Population count (number of set bits).',
  FLO: 'Find leading one.',
  BREV: 'Bit reverse.',
  PRMT: 'Byte permute / shuffle within a register.',

  // --- FP32 ---
  FADD: 'Single-precision floating-point add.',
  FMUL: 'Single-precision floating-point multiply.',
  FFMA: 'Single-precision fused multiply-add.',
  FMNMX: 'Single-precision min / max.',
  FSETP: 'Single-precision compare-and-set predicate.',
  FSET: 'Single-precision compare-and-set register.',
  FCMP: 'Single-precision compare-and-select.',
  FSEL: 'Floating-point select on predicate.',
  FSWZADD: 'Cross-lane swizzle add (used for reductions).',
  FCHK: 'Floating-point check (e.g. divide-by-zero detection).',
  MUFU: 'Multi-function unit: sin, cos, ex2, log2, rcp, rsqrt.',
  RRO: 'Range reduction operation for trig MUFU inputs.',

  // --- FP16 / packed half ---
  HADD2: 'Packed FP16 add (two halves).',
  HMUL2: 'Packed FP16 multiply.',
  HFMA2: 'Packed FP16 fused multiply-add.',
  HMNMX2: 'Packed FP16 min / max.',
  HSETP2: 'Packed FP16 compare-and-set predicate.',
  HSET2: 'Packed FP16 compare-and-set register.',

  // --- FP64 ---
  DADD: 'Double-precision add.',
  DMUL: 'Double-precision multiply.',
  DFMA: 'Double-precision fused multiply-add.',
  DMNMX: 'Double-precision min / max.',
  DSETP: 'Double-precision compare-and-set predicate.',
  DSET: 'Double-precision compare-and-set register.',

  // --- Tensor Core / matrix-multiply-accumulate ---
  HMMA: 'Half-precision matrix multiply-accumulate (Tensor Core).',
  IMMA: 'Integer matrix multiply-accumulate (Tensor Core).',
  BMMA: 'Binary matrix multiply-accumulate (Tensor Core).',
  DMMA: 'Double-precision matrix multiply-accumulate (Tensor Core).',
  QMMA: 'FP8 matrix multiply-accumulate (Tensor Core, Hopper+).',
  TMA: 'Tensor Memory Accelerator transfer (Hopper+).',

  // --- Data movement ---
  MOV: 'Copy a register or immediate into a register.',
  MOV32I: 'Move a 32-bit immediate into a register.',
  SEL: 'Select one of two operands based on a predicate.',
  SHFL: 'Warp shuffle — exchange data between lanes.',
  VOTE: 'Warp vote (all / any / ballot of a predicate).',
  VOTEU: 'Uniform-datapath warp vote.',
  MATCH: 'Find lanes with matching value (Volta+).',

  // --- Memory: generic / global / shared / local / constant ---
  LD: 'Load from generic address space.',
  ST: 'Store to generic address space.',
  LDG: 'Load from global memory.',
  STG: 'Store to global memory.',
  LDS: 'Load from shared memory.',
  STS: 'Store to shared memory.',
  LDL: 'Load from local memory (per-thread spill region).',
  STL: 'Store to local memory.',
  LDC: 'Load from constant memory.',
  ULDC: 'Uniform load from constant memory.',
  ULDG: 'Uniform load from global memory.',
  LDGSTS: 'Async copy from global to shared memory (Ampere+).',
  LDSM: 'Load shared-memory matrix for Tensor Cores.',
  STSM: 'Store shared-memory matrix for Tensor Cores.',
  ATOM: 'Atomic operation in generic address space.',
  ATOMG: 'Atomic operation on global memory.',
  ATOMS: 'Atomic operation on shared memory.',
  RED: 'Memory-side reduction (atomic without return).',
  MEMBAR: 'Memory barrier — order memory operations.',
  ERRBAR: 'Error barrier — flush async errors.',
  CCTL: 'Cache control hint for global memory.',
  CCTLL: 'Cache control hint for local memory.',

  // --- Texture / surface ---
  TEX: 'Texture lookup.',
  TLD: 'Texel load (untextured).',
  TLD4: 'Texture gather-4.',
  TXD: 'Texture lookup with explicit derivatives.',
  TXQ: 'Texture query (size, levels, etc.).',
  SULD: 'Surface load.',
  SUST: 'Surface store.',
  SURED: 'Surface reduction.',

  // --- Control flow ---
  BRA: 'Branch (conditional or unconditional).',
  BRX: 'Indexed branch (jump table).',
  BMOV: 'Branch via register-stored target.',
  BSSY: 'Push a sync target onto the convergence barrier stack.',
  BSYNC: 'Pop / sync to a convergence barrier (Volta+).',
  BPT: 'Breakpoint trap.',
  CALL: 'Function call.',
  RET: 'Function return.',
  JMP: 'Unconditional jump.',
  JMX: 'Indexed jump.',
  KILL: 'Terminate the current thread.',
  EXIT: 'Thread exits the kernel.',
  PEXIT: 'Predicated early exit.',
  NANOSLEEP: '__nanosleep(): suspend the warp for a short interval.',
  YIELD: 'Hint that the warp should yield the SM.',
  TRAP: 'Software trap (debugger / abort).',
  BAR: 'Block-level barrier (e.g. __syncthreads).',
  WARPSYNC: 'Warp-level synchronization (Volta+).',
  DEPBAR: 'Dependency barrier on previous async memory ops.',

  // --- Predicate / uniform helpers ---
  PSETP: 'Set predicate from logical op on two predicates.',
  P2R: 'Move predicate(s) into a general register.',
  R2P: 'Move bits from a register into predicate registers.',
  CSET: 'Conditional code set.',
  CSETP: 'Conditional code set predicate.',

  // --- Conversion ---
  F2I: 'Float to integer convert.',
  I2F: 'Integer to float convert.',
  F2F: 'Float to float convert (precision change).',
  I2I: 'Integer to integer convert (sign / zero extend / truncate).',
  FRND: 'Round float to integer-valued float.',

  // --- Special registers / misc ---
  S2R: 'Read a special register (threadIdx, laneid, clock, ...).',
  CS2R: 'Coupled special-register read (64-bit).',
  R2UR: 'Move register value to uniform register.',
  S2UR: 'Special-register read into uniform register.',
  B2R: 'Read a barrier state into a register.',
  NOP: 'No operation.',

  // --- Uniform datapath (Turing+) ---
  UMOV: 'Move on the uniform datapath.',
  UIADD3: 'Three-operand integer add on the uniform datapath.',
  UIMAD: 'Integer multiply-add on the uniform datapath.',
  UISETP: 'Integer compare-and-set predicate on the uniform datapath.',
  ULOP3: 'Three-input bitwise logic on the uniform datapath.',
  USHF: 'Funnel shift on the uniform datapath.',
  UBMSK: 'Build a mask on the uniform datapath.',
  USEL: 'Select on the uniform datapath.',
};

export const STALL_REASON_DESCRIPTIONS: Record<string, string> = {
  // --- Eligible / issued (not actually stalled) ---
  smsp__pcsamp_warps_issue_selected:
    'Warp was selected by the scheduler and issued this cycle. Good — counts as forward progress.',
  smsp__pcsamp_warps_issue_not_selected:
    'Warp was eligible but the scheduler picked a different warp. Often benign on occupied SMs.',

  // --- Memory / scoreboard ---
  smsp__pcsamp_warps_issue_stalled_long_scoreboard:
    'Waiting on a memory dependency — typically a global or local load. Most common stall on memory-bound kernels.',
  smsp__pcsamp_warps_issue_stalled_short_scoreboard:
    'Waiting on an MIO pipe dependency — typically shared memory or special-function-unit result.',

  // --- Fixed-latency dependency ---
  smsp__pcsamp_warps_issue_stalled_wait:
    'Waiting on a fixed-latency execution dependency from a recent math instruction.',

  // --- Throttles (queues full) ---
  smsp__pcsamp_warps_issue_stalled_math_pipe_throttle:
    'Math pipeline is saturated — too many in-flight instructions of the same type.',
  smsp__pcsamp_warps_issue_stalled_mio_throttle:
    'MIO instruction queue full — shared / special-function / load-store ops backed up.',
  smsp__pcsamp_warps_issue_stalled_lg_throttle: 'Local / global memory instruction queue full.',
  smsp__pcsamp_warps_issue_stalled_tex_throttle: 'Texture unit instruction queue full.',

  // --- Synchronization ---
  smsp__pcsamp_warps_issue_stalled_barrier:
    'Waiting at a __syncthreads() barrier for other warps in the block.',
  smsp__pcsamp_warps_issue_stalled_membar:
    'Waiting on a memory barrier (__threadfence) for outstanding memory ops to commit.',
  smsp__pcsamp_warps_issue_stalled_sync: 'Waiting on warp-level sync (e.g. __syncwarp).',

  // --- Front-end / dispatch ---
  smsp__pcsamp_warps_issue_stalled_branch_resolving: 'Waiting for a branch target to resolve.',
  smsp__pcsamp_warps_issue_stalled_dispatch_stall:
    'Dispatcher could not issue this cycle due to resource contention.',
  smsp__pcsamp_warps_issue_stalled_no_instruction:
    'No instruction available to issue — typically an instruction-cache miss.',
  smsp__pcsamp_warps_issue_stalled_imc_miss: 'Waiting on an instruction-cache miss.',

  // --- Lifecycle ---
  smsp__pcsamp_warps_issue_stalled_drain: 'Warp draining instructions before exit. Not actionable.',
  smsp__pcsamp_warps_issue_stalled_sleeping: 'Warp is executing __nanosleep().',
  smsp__pcsamp_warps_issue_stalled_selected:
    'Selected and issued this cycle (counts running warps; this is "good").',
  smsp__pcsamp_warps_issue_stalled_not_selected:
    'Eligible but another warp was selected by the scheduler.',
  smsp__pcsamp_warps_issue_stalled_allocation:
    'Waiting on register / shared-memory allocation at warp launch.',
  smsp__pcsamp_warps_issue_stalled_work_steal:
    'Idle slot waiting for new work to be scheduled (programmatic dependent launch).',
  smsp__pcsamp_warps_issue_stalled_limit:
    'Hit a hardware-imposed in-flight limit (e.g. outstanding memory transactions).',
  smsp__pcsamp_warps_issue_stalled_sb_full:
    "Scoreboard tracking structure is full — can't track more outstanding ops.",
  smsp__pcsamp_warps_issue_stalled_idx_throttle: 'Indexed-constant load queue full.',
  smsp__pcsamp_warps_issue_stalled_misc:
    "Catch-all for stalls that don't fall into a specific category.",

  // --- Uniform datapath specific (Turing+) ---
  smsp__pcsamp_warps_issue_stalled_udp_throttle: 'Uniform datapath instruction queue full.',

  // --- Hopper / async copy specific ---
  smsp__pcsamp_warps_issue_stalled_tma_throttle: 'Tensor Memory Accelerator queue full (Hopper+).',
  smsp__pcsamp_warps_issue_stalled_cga_barrier:
    'Waiting at a cluster (CGA) barrier for cooperating blocks (Hopper+).',
};

export function gpuFrameDescription(name: string): string | undefined {
  if (name === '') return undefined;
  return SASS_INSTRUCTION_DESCRIPTIONS[name] ?? STALL_REASON_DESCRIPTIONS[name];
}
