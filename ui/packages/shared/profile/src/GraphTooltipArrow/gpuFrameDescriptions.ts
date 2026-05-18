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
//      `.FTZ` stripped). Source: NVIDIA CUDA Binary Utilities · cuobjdump
//      SASS reference, plus the opcode tables in
//      https://github.com/gnurizen/sass-table .
//
//   2. CUPTI / Nsight Compute warp PC-sampling reasons — lower-case metric
//      names of the shape `smsp__pcsamp_warps_issue_<state>`. Source:
//      NVIDIA Nsight Compute · Warp Stall Reasons.
//
// The Description field, when present, is intended to carry NVIDIA's prose
// VERBATIM (no summarization). Entries with shorter blurbs below are
// placeholders that should be replaced with the canonical NVIDIA text as
// reviewers fill them in.
//
// Both lookups are exact-match on the frame name; no risk of false positives
// on user function names.

export const NVIDIA_DOCS_LABEL = 'NVIDIA docs';
export const STALL_SOURCE_URL =
  'https://docs.nvidia.com/nsight-compute/ProfilingGuide/index.html#warp-stall-reasons';
export const SASS_SOURCE_URL =
  'https://docs.nvidia.com/cuda/cuda-binary-utilities/index.html#turing-turing-instruction-set-table';

export interface StallEntry {
  reasonLabel: string;
  description: string;
  sourceUrl?: string;
}

export interface SASSEntry {
  reasonLabel: string;
  description: string;
  sourceUrl?: string;
}

// Ref: https://docs.nvidia.com/cuda/cuda-binary-utilities/index.html#turing-turing-instruction-set-table
export const SASS_INSTRUCTION_DESCRIPTIONS: Record<string, SASSEntry> = {
  // --- Floating Point Instructions ---
  FADD: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Add'},
  FCHK: {reasonLabel: 'Floating Point Instructions', description: 'Floating-point Range Check'},
  FFMA: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Fused Multiply and Add'},
  FMNMX: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Minimum/Maximum'},
  FMUL: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Multiply'},
  FSEL: {reasonLabel: 'Floating Point Instructions', description: 'Floating Point Select'},
  FSET: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Compare And Set'},
  FSETP: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Compare And Set Predicate'},
  FSWZADD: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Swizzle Add'},
  MUFU: {reasonLabel: 'Floating Point Instructions', description: 'FP32 Multi Function Operation'},
  HADD2: {reasonLabel: 'Floating Point Instructions', description: 'FP16 Add'},
  HFMA2: {reasonLabel: 'Floating Point Instructions', description: 'FP16 Fused Mutiply Add'},
  HMMA: {reasonLabel: 'Floating Point Instructions', description: 'Matrix Multiply and Accumulate'},
  HMUL2: {reasonLabel: 'Floating Point Instructions', description: 'FP16 Multiply'},
  HSET2: {reasonLabel: 'Floating Point Instructions', description: 'FP16 Compare And Set'},
  HSETP2: {reasonLabel: 'Floating Point Instructions', description: 'FP16 Compare And Set Predicate'},
  DADD: {reasonLabel: 'Floating Point Instructions', description: 'FP64 Add'},
  DFMA: {reasonLabel: 'Floating Point Instructions', description: 'FP64 Fused Mutiply Add'},
  DMUL: {reasonLabel: 'Floating Point Instructions', description: 'FP64 Multiply'},
  DSETP: {reasonLabel: 'Floating Point Instructions', description: 'FP64 Compare And Set Predicate'},

  // --- Integer Instructions ---
  BMMA: {reasonLabel: 'Integer Instructions', description: 'Bit Matrix Multiply and Accumulate'},
  BMSK: {reasonLabel: 'Integer Instructions', description: 'Bitfield Mask'},
  BREV: {reasonLabel: 'Integer Instructions', description: 'Bit Reverse'},
  FLO: {reasonLabel: 'Integer Instructions', description: 'Find Leading One'},
  IABS: {reasonLabel: 'Integer Instructions', description: 'Integer Absolute Value'},
  IADD: {reasonLabel: 'Integer Instructions', description: 'Integer Addition'},
  IADD3: {reasonLabel: 'Integer Instructions', description: '3-input Integer Addition'},
  IDP: {reasonLabel: 'Integer Instructions', description: 'Integer Dot Product and Accumulate'},
  IDP4A: {reasonLabel: 'Integer Instructions', description: 'Integer Dot Product and Accumulate'},
  IMAD: {reasonLabel: 'Integer Instructions', description: 'Integer Multiply And Add'},
  IMMA: {reasonLabel: 'Integer Instructions', description: 'Integer Matrix Multiply and Accumulate'},
  IMNMX: {reasonLabel: 'Integer Instructions', description: 'Integer Minimum/Maximum'},
  IMUL: {reasonLabel: 'Integer Instructions', description: 'Integer Multiply'},
  ISCADD: {reasonLabel: 'Integer Instructions', description: 'Scaled Integer Addition'},
  ISETP: {reasonLabel: 'Integer Instructions', description: 'Integer Compare And Set Predicate'},
  LEA: {reasonLabel: 'Integer Instructions', description: 'LOAD Effective Address'},
  LOP: {reasonLabel: 'Integer Instructions', description: 'Logic Operation'},
  LOP3: {reasonLabel: 'Integer Instructions', description: 'Logic Operation'},
  POPC: {reasonLabel: 'Integer Instructions', description: 'Population count'},
  SHF: {reasonLabel: 'Integer Instructions', description: 'Funnel Shift'},
  SHL: {reasonLabel: 'Integer Instructions', description: 'Shift Left'},
  SHR: {reasonLabel: 'Integer Instructions', description: 'Shift Right'},
  VABSDIFF: {reasonLabel: 'Integer Instructions', description: 'Absolute Difference'},
  VABSDIFF4: {reasonLabel: 'Integer Instructions', description: 'Absolute Difference'},

  // --- Conversion Instructions ---
  F2F: {reasonLabel: 'Conversion Instructions', description: 'Floating Point To Floating Point Conversion'},
  F2I: {reasonLabel: 'Conversion Instructions', description: 'Floating Point To Integer Conversion'},
  I2F: {reasonLabel: 'Conversion Instructions', description: 'Integer To Floating Point Conversion'},
  I2I: {reasonLabel: 'Conversion Instructions', description: 'Integer To Integer Conversion'},
  I2IP: {reasonLabel: 'Conversion Instructions', description: 'Integer To Integer Conversion and Packing'},
  FRND: {reasonLabel: 'Conversion Instructions', description: 'Round To Integer'},

  // --- Movement Instructions ---
  MOV: {reasonLabel: 'Movement Instructions', description: 'Move'},
  MOVM: {reasonLabel: 'Movement Instructions', description: 'Move Matrix with Transposition or Expansion'},
  PRMT: {reasonLabel: 'Movement Instructions', description: 'Permute Register Pair'},
  SEL: {reasonLabel: 'Movement Instructions', description: 'Select Source with Predicate'},
  SGXT: {reasonLabel: 'Movement Instructions', description: 'Sign Extend'},
  SHFL: {reasonLabel: 'Movement Instructions', description: 'Warp Wide Register Shuffle'},

  // --- Predicate Instructions ---
  PLOP3: {reasonLabel: 'Predicate Instructions', description: 'Predicate Logic Operation'},
  PSETP: {reasonLabel: 'Predicate Instructions', description: 'Combine Predicates and Set Predicate'},
  P2R: {reasonLabel: 'Predicate Instructions', description: 'Move Predicate Register To Register'},
  R2P: {reasonLabel: 'Predicate Instructions', description: 'Move Register To Predicate Register'},

  // --- Load/Store Instructions ---
  LD: {reasonLabel: 'Load/Store Instructions', description: 'Load from generic Memory'},
  LDC: {reasonLabel: 'Load/Store Instructions', description: 'Load Constant'},
  LDG: {reasonLabel: 'Load/Store Instructions', description: 'Load from Global Memory'},
  LDL: {reasonLabel: 'Load/Store Instructions', description: 'Load within Local Memory Window'},
  LDS: {reasonLabel: 'Load/Store Instructions', description: 'Load within Shared Memory Window'},
  LDSM: {reasonLabel: 'Load/Store Instructions', description: 'Load Matrix from Shared Memory with Element Size Expansion'},
  ST: {reasonLabel: 'Load/Store Instructions', description: 'Store to Generic Memory'},
  STG: {reasonLabel: 'Load/Store Instructions', description: 'Store to Global Memory'},
  STL: {reasonLabel: 'Load/Store Instructions', description: 'Store to Local Memory'},
  STS: {reasonLabel: 'Load/Store Instructions', description: 'Store to Shared Memory'},
  MATCH: {reasonLabel: 'Load/Store Instructions', description: 'Match Register Values Across Thread Group'},
  QSPC: {reasonLabel: 'Load/Store Instructions', description: 'Query Space'},
  ATOM: {reasonLabel: 'Load/Store Instructions', description: 'Atomic Operation on Generic Memory'},
  ATOMS: {reasonLabel: 'Load/Store Instructions', description: 'Atomic Operation on Shared Memory'},
  ATOMG: {reasonLabel: 'Load/Store Instructions', description: 'Atomic Operation on Global Memory'},
  RED: {reasonLabel: 'Load/Store Instructions', description: 'Reduction Operation on Generic Memory'},
  CCTL: {reasonLabel: 'Load/Store Instructions', description: 'Cache Control'},
  CCTLL: {reasonLabel: 'Load/Store Instructions', description: 'Cache Control'},
  ERRBAR: {reasonLabel: 'Load/Store Instructions', description: 'Error Barrier'},
  MEMBAR: {reasonLabel: 'Load/Store Instructions', description: 'Memory Barrier'},
  CCTLT: {reasonLabel: 'Load/Store Instructions', description: 'Texture Cache Control'},

  // --- Uniform Datapath Instructions ---
  R2UR: {reasonLabel: 'Uniform Datapath Instructions', description: 'Move from Vector Register to a Uniform Register'},
  S2UR: {reasonLabel: 'Uniform Datapath Instructions', description: 'Move Special Register to Uniform Register'},
  UBMSK: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Bitfield Mask'},
  UBREV: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Bit Reverse'},
  UCLEA: {reasonLabel: 'Uniform Datapath Instructions', description: 'Load Effective Address for a Constant'},
  UFLO: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Find Leading One'},
  UIADD3: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Integer Addition'},
  UIMAD: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Integer Multiplication'},
  UISETP: {reasonLabel: 'Uniform Datapath Instructions', description: 'Integer Compare and Set Uniform Predicate'},
  ULDC: {reasonLabel: 'Uniform Datapath Instructions', description: 'Load from Constant Memory into a Uniform Register'},
  ULEA: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Load Effective Address'},
  ULOP: {reasonLabel: 'Uniform Datapath Instructions', description: 'Logic Operation'},
  ULOP3: {reasonLabel: 'Uniform Datapath Instructions', description: 'Logic Operation'},
  UMOV: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Move'},
  UP2UR: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Predicate to Uniform Register'},
  UPLOP3: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Predicate Logic Operation'},
  UPOPC: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Population Count'},
  UPRMT: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Byte Permute'},
  UPSETP: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Predicate Logic Operation'},
  UR2UP: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Register to Uniform Predicate'},
  USEL: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Select'},
  USGXT: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Sign Extend'},
  USHF: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Funnel Shift'},
  USHL: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Left Shift'},
  USHR: {reasonLabel: 'Uniform Datapath Instructions', description: 'Uniform Right Shift'},
  VOTEU: {reasonLabel: 'Uniform Datapath Instructions', description: 'Voting across SIMD Thread Group with Results in Uniform Destination'},

  // --- Texture Instructions ---
  TEX: {reasonLabel: 'Texture Instructions', description: 'Texture Fetch'},
  TLD: {reasonLabel: 'Texture Instructions', description: 'Texture Load'},
  TLD4: {reasonLabel: 'Texture Instructions', description: 'Texture Load 4'},
  TMML: {reasonLabel: 'Texture Instructions', description: 'Texture MipMap Level'},
  TXD: {reasonLabel: 'Texture Instructions', description: 'Texture Fetch With Derivatives'},
  TXQ: {reasonLabel: 'Texture Instructions', description: 'Texture Query'},

  // --- Surface Instructions ---
  SUATOM: {reasonLabel: 'Surface Instructions', description: 'Atomic Op on Surface Memory'},
  SULD: {reasonLabel: 'Surface Instructions', description: 'Surface Load'},
  SURED: {reasonLabel: 'Surface Instructions', description: 'Reduction Op on Surface Memory'},
  SUST: {reasonLabel: 'Surface Instructions', description: 'Surface Store'},

  // --- Control Instructions ---
  BMOV: {reasonLabel: 'Control Instructions', description: 'Move Convergence Barrier State'},
  BPT: {reasonLabel: 'Control Instructions', description: 'BreakPoint/Trap'},
  BRA: {reasonLabel: 'Control Instructions', description: 'Relative Branch'},
  BREAK: {reasonLabel: 'Control Instructions', description: 'Break out of the Specified Convergence Barrier'},
  BRX: {reasonLabel: 'Control Instructions', description: 'Relative Branch Indirect'},
  BRXU: {reasonLabel: 'Control Instructions', description: 'Relative Branch with Uniform Register Based Offset'},
  BSSY: {reasonLabel: 'Control Instructions', description: 'Barrier Set Convergence Synchronization Point'},
  BSYNC: {reasonLabel: 'Control Instructions', description: 'Synchronize Threads on a Convergence Barrier'},
  CALL: {reasonLabel: 'Control Instructions', description: 'Call Function'},
  EXIT: {reasonLabel: 'Control Instructions', description: 'Exit Program'},
  JMP: {reasonLabel: 'Control Instructions', description: 'Absolute Jump'},
  JMX: {reasonLabel: 'Control Instructions', description: 'Absolute Jump Indirect'},
  JMXU: {reasonLabel: 'Control Instructions', description: 'Absolute Jump with Uniform Register Based Offset'},
  KILL: {reasonLabel: 'Control Instructions', description: 'Kill Thread'},
  NANOSLEEP: {reasonLabel: 'Control Instructions', description: 'Suspend Execution'},
  RET: {reasonLabel: 'Control Instructions', description: 'Return From Subroutine'},
  RPCMOV: {reasonLabel: 'Control Instructions', description: 'PC Register Move'},
  RTT: {reasonLabel: 'Control Instructions', description: 'Return From Trap'},
  WARPSYNC: {reasonLabel: 'Control Instructions', description: 'Synchronize Threads in Warp'},
  YIELD: {reasonLabel: 'Control Instructions', description: 'Yield Control'},

  // --- Miscellaneous Instructions ---
  B2R: {reasonLabel: 'Miscellaneous Instructions', description: 'Move Barrier To Register'},
  BAR: {reasonLabel: 'Miscellaneous Instructions', description: 'Barrier Synchronization'},
  CS2R: {reasonLabel: 'Miscellaneous Instructions', description: 'Move Special Register to Register'},
  DEPBAR: {reasonLabel: 'Miscellaneous Instructions', description: 'Dependency Barrier'},
  GETLMEMBASE: {reasonLabel: 'Miscellaneous Instructions', description: 'Get Local Memory Base Address'},
  LEPC: {reasonLabel: 'Miscellaneous Instructions', description: 'Load Effective PC'},
  NOP: {reasonLabel: 'Miscellaneous Instructions', description: 'No Operation'},
  PMTRIG: {reasonLabel: 'Miscellaneous Instructions', description: 'Performance Monitor Trigger'},
  R2B: {reasonLabel: 'Miscellaneous Instructions', description: 'Move Register to Barrier'},
  S2R: {reasonLabel: 'Miscellaneous Instructions', description: 'Move Special Register to Register'},
  SETCTAID: {reasonLabel: 'Miscellaneous Instructions', description: 'Set CTA ID'},
  SETLMEMBASE: {reasonLabel: 'Miscellaneous Instructions', description: 'Set Local Memory Base Address'},
  VOTE: {reasonLabel: 'Miscellaneous Instructions', description: 'Vote Across SIMD Thread Group'},
};

// Ref: https://docs.nvidia.com/nsight-compute/ProfilingGuide/index.html#warp-stall-reasons
export const STALL_REASON_DESCRIPTIONS: Record<string, StallEntry> = {
  // --- Warp Stall Reasons ---
  smsp__pcsamp_warps_issue_stalled_barrier: {
    reasonLabel: 'Barrier',
    description:
      'Warp stalled waiting for sibling warps to reach a CTA barrier. This is usually caused by divergent code paths before the barrier, which make some warps wait a long time while others catch up to the synchronization point. Where possible, split work into uniform-sized blocks; for blocks of 512 threads or more, consider breaking them into smaller groups. That raises the number of eligible warps without changing occupancy, unless shared memory then becomes the occupancy limiter. It also helps to identify which barrier instruction stalls the most and optimize the code that runs before that synchronization point first.',
  },
  smsp__pcsamp_warps_issue_stalled_branch_resolving: {
    reasonLabel: 'Branch Resolving',
    description:
      'Warp stalled waiting for a branch target to be computed and the warp program counter to be updated. Cut stalled cycles by using fewer jump/branch operations and reducing control-flow divergence, e.g. by reducing or coalescing conditionals in the code. See also the related No Instructions state.',
  },
  smsp__pcsamp_warps_issue_stalled_dispatch_stall: {
    reasonLabel: 'Dispatch Stall',
    description:
      'Warp stalled on a dispatch stall: the warp has an instruction ready to issue, but the dispatcher withholds it because of other conflicts or events.',
  },
  smsp__pcsamp_warps_issue_stalled_drain: {
    reasonLabel: 'Drain',
    description:
      'Warp stalled after EXIT, waiting for all outstanding memory operations to complete so the warp’s resources can be freed. A high number of draining warps typically happens when a lot of data is written to memory towards the end of a kernel. Make sure those store operations use memory access patterns optimal for the target architecture, and consider a parallelized data reduction where applicable.',
  },
  smsp__pcsamp_warps_issue_stalled_imc_miss: {
    reasonLabel: 'IMC Miss',
    description:
      'Warp stalled on an immediate constant cache (IMC) miss. A read from constant memory costs one device-memory read only on a cache miss; otherwise it costs just one read from the constant cache. Immediate constants are encoded into the SASS instruction as ‘c[bank][offset]’. Accesses to different addresses by threads within a warp are serialized, so the cost scales linearly with the number of unique addresses the warp reads. The constant cache is therefore best when threads in the same warp access only a few distinct locations; if all threads of a warp access the same location, constant memory can be as fast as a register access.',
  },
  smsp__pcsamp_warps_issue_stalled_lg_throttle: {
    reasonLabel: 'LG Throttle',
    description:
      'Warp stalled waiting for the L1 instruction queue for local and global (LG) memory operations to be not full. This typically occurs only when local or global memory instructions execute extremely frequently. Avoid redundant global memory accesses. Try to avoid thread-local memory by checking whether dynamically indexed arrays are declared in local scope, or whether excessive register pressure is causing spills. Where applicable, combine multiple lower-width memory operations into fewer wider ones and interleave memory operations with math instructions.',
  },
  smsp__pcsamp_warps_issue_stalled_long_scoreboard: {
    reasonLabel: 'Long Scoreboard',
    description:
      'Warp stalled on a scoreboard dependency for an L1TEX (local, global, surface, texture) operation. Find the instruction producing the awaited data to identify the cause. To reduce cycles spent waiting on L1TEX accesses, verify that memory access patterns are optimal for the target architecture, raise cache hit rates by improving data locality (coalescing) or by changing the cache configuration, and consider moving frequently used data into shared memory.',
  },
  smsp__pcsamp_warps_issue_stalled_math_pipe_throttle: {
    reasonLabel: 'Math Pipe Throttle',
    description:
      'Warp stalled waiting for an execution pipe to become available. This happens when every active warp’s next instruction targets the same oversubscribed math pipeline. Increase the number of active warps to hide the latency, or change the instruction mix to use all available pipelines more evenly.',
  },
  smsp__pcsamp_warps_issue_stalled_membar: {
    reasonLabel: 'Membar',
    description:
      'Warp stalled on a memory barrier. Avoid any unnecessary memory barriers and make sure outstanding memory operations are fully optimized for the target architecture.',
  },
  smsp__pcsamp_warps_issue_stalled_mio_throttle: {
    reasonLabel: 'MIO Throttle',
    description:
      'Warp stalled waiting for the MIO (memory input/output) instruction queue to be not full. This is high under extreme utilization of the MIO pipelines, which include special math instructions, dynamic branches, and shared memory instructions. When the cause is shared memory accesses, using fewer but wider loads can reduce the pipeline pressure.',
  },
  smsp__pcsamp_warps_issue_stalled_misc: {
    reasonLabel: 'Misc',
    description: 'Warp stalled for a miscellaneous hardware reason.',
  },
  smsp__pcsamp_warps_issue_stalled_no_instructions: {
    reasonLabel: 'No Instructions',
    description:
      'Warp stalled waiting to be selected to fetch an instruction, or on an instruction cache miss. Many warps in this state is typical for very short kernels with less than one full wave of work in the grid. Excessively jumping across large blocks of assembly code can also cause it, if that misses in the instruction cache. See also the related Branch Resolving state.',
  },
  smsp__pcsamp_warps_issue_stalled_not_selected: {
    reasonLabel: 'Not Selected',
    description:
      'Warp stalled waiting for the micro scheduler to select it to issue. Not-selected warps are eligible warps that were not picked that cycle because another warp was selected instead. A high number typically means you have enough warps to cover warp latencies, and you may be able to reduce the number of active warps to improve cache coherence and data locality.',
  },
  smsp__pcsamp_warps_issue_stalled_selected: {
    reasonLabel: 'Selected',
    description: 'Warp was selected by the micro scheduler and issued an instruction.',
  },
  smsp__pcsamp_warps_issue_stalled_short_scoreboard: {
    reasonLabel: 'Short Scoreboard',
    description:
      'Warp stalled on a scoreboard dependency for a MIO (memory input/output) operation that is not to L1TEX. The primary cause is usually shared memory operations; other causes include frequent special math instructions (e.g. MUFU) or dynamic branching (e.g. BRX, JMX). Check the Memory Workload Analysis section for shared memory operations and reduce any reported bank conflicts. Assigning frequently accessed values to variables can help the compiler use low-latency registers instead of direct memory accesses.',
  },
  smsp__pcsamp_warps_issue_stalled_sleeping: {
    reasonLabel: 'Sleeping',
    description:
      'Warp stalled because all of its threads are in the blocked, yielded, or sleep state. Reduce the number of NANOSLEEP instructions executed, lower the specified time delay, and try to arrange for multiple threads in a warp to sleep at the same time.',
  },
  smsp__pcsamp_warps_issue_stalled_tex_throttle: {
    reasonLabel: 'Tex Throttle',
    description:
      'Warp stalled waiting for the L1 instruction queue for texture operations to be not full. This is high under extreme utilization of the L1TEX pipeline. Issue fewer texture fetches, surface loads, surface stores, or decoupled math operations. Where applicable, combine multiple lower-width memory operations into fewer wider ones and interleave memory operations with math instructions. Consider converting texture lookups or surface loads into global memory lookups: texture accepts four threads’ requests per cycle, whereas global accepts 32 threads.',
  },
  smsp__pcsamp_warps_issue_stalled_wait: {
    reasonLabel: 'Wait',
    description:
      'Warp stalled on a fixed-latency execution dependency. This should normally be very low and only shows up as a top contributor in already highly optimized kernels. Hide the instruction latencies by increasing the number of active warps, restructuring the code, or unrolling loops; you can also switch to lower-latency instructions, e.g. via fast-math compiler options.',
  },
  smsp__pcsamp_warps_issue_stalled_warpgroup_arrive: {
    reasonLabel: 'Warpgroup Arrive',
    description: 'Warp stalled waiting on a WARPGROUP.ARRIVES or WARPGROUP.WAIT instruction.',
  },

  // --- Warp Stall Reasons (Not Issued) ---
  smsp__pcsamp_warps_issue_stalled_barrier_not_issued: {
    reasonLabel: 'Barrier (Not Issued)',
    description:
      'Warp stalled waiting for sibling warps to reach a CTA barrier. This is usually caused by divergent code paths before the barrier, which make some warps wait a long time while others catch up to the synchronization point. Where possible, split work into uniform-sized blocks; for blocks of 512 threads or more, consider breaking them into smaller groups. That raises the number of eligible warps without changing occupancy, unless shared memory then becomes the occupancy limiter. It also helps to identify which barrier instruction stalls the most and optimize the code that runs before that synchronization point first.',
  },
  smsp__pcsamp_warps_issue_stalled_branch_resolving_not_issued: {
    reasonLabel: 'Branch Resolving (Not Issued)',
    description:
      'Warp stalled waiting for a branch target to be computed and the warp program counter to be updated. Cut stalled cycles by using fewer jump/branch operations and reducing control-flow divergence, e.g. by reducing or coalescing conditionals in the code. See also the related No Instructions state.',
  },
  smsp__pcsamp_warps_issue_stalled_dispatch_stall_not_issued: {
    reasonLabel: 'Dispatch Stall (Not Issued)',
    description:
      'Warp stalled on a dispatch stall: the warp has an instruction ready to issue, but the dispatcher withholds it because of other conflicts or events.',
  },
  smsp__pcsamp_warps_issue_stalled_drain_not_issued: {
    reasonLabel: 'Drain (Not Issued)',
    description:
      'Warp stalled after EXIT, waiting for all memory operations to complete so warp resources can be freed. A high number of draining warps typically happens when a lot of data is written to memory towards the end of a kernel. Make sure those store operations use memory access patterns optimal for the target architecture, and consider a parallelized data reduction where applicable.',
  },
  smsp__pcsamp_warps_issue_stalled_imc_miss_not_issued: {
    reasonLabel: 'IMC Miss (Not Issued)',
    description:
      'Warp stalled on an immediate constant cache (IMC) miss. A read from constant memory costs one device-memory read only on a cache miss; otherwise it costs just one read from the constant cache. Accesses to different addresses by threads within a warp are serialized, so the cost scales linearly with the number of unique addresses the warp reads. The constant cache is therefore best when threads in the same warp access only a few distinct locations; if all threads of a warp access the same location, constant memory can be as fast as a register access.',
  },
  smsp__pcsamp_warps_issue_stalled_lg_throttle_not_issued: {
    reasonLabel: 'LG Throttle (Not Issued)',
    description:
      'Warp stalled waiting for the L1 instruction queue for local and global (LG) memory operations to be not full. This typically occurs only when local or global memory instructions execute extremely frequently. Avoid redundant global memory accesses. Try to avoid thread-local memory by checking whether dynamically indexed arrays are declared in local scope, or whether excessive register pressure is causing spills. Where applicable, combine multiple lower-width memory operations into fewer wider ones and interleave memory operations with math instructions.',
  },
  smsp__pcsamp_warps_issue_stalled_long_scoreboard_not_issued: {
    reasonLabel: 'Long Scoreboard (Not Issued)',
    description:
      'Warp stalled on a scoreboard dependency for an L1TEX (local, global, surface, texture) operation. Find the instruction producing the awaited data to identify the cause. To reduce cycles spent waiting on L1TEX accesses, verify that memory access patterns are optimal for the target architecture, raise cache hit rates by improving data locality (coalescing) or by changing the cache configuration, and consider moving frequently used data into shared memory.',
  },
  smsp__pcsamp_warps_issue_stalled_math_pipe_throttle_not_issued: {
    reasonLabel: 'Math Pipe Throttle (Not Issued)',
    description:
      'Warp stalled waiting for an execution pipe to become available. This happens when every active warp’s next instruction targets the same oversubscribed math pipeline. Increase the number of active warps to hide the latency, or change the instruction mix to use all available pipelines more evenly.',
  },
  smsp__pcsamp_warps_issue_stalled_membar_not_issued: {
    reasonLabel: 'Membar (Not Issued)',
    description:
      'Warp stalled on a memory barrier. Avoid any unnecessary memory barriers and make sure outstanding memory operations are fully optimized for the target architecture.',
  },
  smsp__pcsamp_warps_issue_stalled_mio_throttle_not_issued: {
    reasonLabel: 'MIO Throttle (Not Issued)',
    description:
      'Warp stalled waiting for the MIO (memory input/output) instruction queue to be not full. This is high under extreme utilization of the MIO pipelines, which include special math instructions, dynamic branches, and shared memory instructions. When the cause is shared memory accesses, using fewer but wider loads can reduce the pipeline pressure.',
  },
  smsp__pcsamp_warps_issue_stalled_misc_not_issued: {
    reasonLabel: 'Misc (Not Issued)',
    description: 'Warp stalled for a miscellaneous hardware reason.',
  },
  smsp__pcsamp_warps_issue_stalled_no_instructions_not_issued: {
    reasonLabel: 'No Instructions (Not Issued)',
    description:
      'Warp stalled waiting to be selected to fetch an instruction, or on an instruction cache miss. Many warps in this state is typical for very short kernels with less than one full wave of work in the grid. Excessively jumping across large blocks of assembly code can also cause it, if that misses in the instruction cache. See also the related Branch Resolving state.',
  },
  smsp__pcsamp_warps_issue_stalled_not_selected_not_issued: {
    reasonLabel: 'Not Selected (Not Issued)',
    description:
      'Warp stalled waiting for the micro scheduler to select it to issue. Not-selected warps are eligible warps that were not picked that cycle because another warp was selected instead. A high number typically means you have enough warps to cover warp latencies, and you may be able to reduce the number of active warps to improve cache coherence and data locality.',
  },
  smsp__pcsamp_warps_issue_stalled_selected_not_issued: {
    reasonLabel: 'Selected (Not Issued)',
    description: 'Warp was selected by the micro scheduler and issued an instruction.',
  },
  smsp__pcsamp_warps_issue_stalled_short_scoreboard_not_issued: {
    reasonLabel: 'Short Scoreboard (Not Issued)',
    description:
      'Warp stalled on a scoreboard dependency for a MIO (memory input/output) operation that is not to L1TEX. The primary cause is usually shared memory operations; other causes include frequent special math instructions (e.g. MUFU) or dynamic branching (e.g. BRX, JMX). Check the Memory Workload Analysis section for shared memory operations and reduce any reported bank conflicts. Assigning frequently accessed values to variables can help the compiler use low-latency registers instead of direct memory accesses.',
  },
  smsp__pcsamp_warps_issue_stalled_sleeping_not_issued: {
    reasonLabel: 'Sleeping (Not Issued)',
    description:
      'Warp stalled because all of its threads are in the blocked, yielded, or sleep state. Reduce the number of NANOSLEEP instructions executed, lower the specified time delay, and try to arrange for multiple threads in a warp to sleep at the same time.',
  },
  smsp__pcsamp_warps_issue_stalled_tex_throttle_not_issued: {
    reasonLabel: 'Tex Throttle (Not Issued)',
    description:
      'Warp stalled waiting for the L1 instruction queue for texture operations to be not full. This is high under extreme utilization of the L1TEX pipeline. Issue fewer texture fetches, surface loads, surface stores, or decoupled math operations. Where applicable, combine multiple lower-width memory operations into fewer wider ones and interleave memory operations with math instructions. Consider converting texture lookups or surface loads into global memory lookups: texture accepts four threads’ requests per cycle, whereas global accepts 32 threads.',
  },
  smsp__pcsamp_warps_issue_stalled_wait_not_issued: {
    reasonLabel: 'Wait (Not Issued)',
    description:
      'Warp stalled on a fixed-latency execution dependency. This should normally be very low and only shows up as a top contributor in already highly optimized kernels. Hide the instruction latencies by increasing the number of active warps, restructuring the code, or unrolling loops; you can also switch to lower-latency instructions, e.g. via fast-math compiler options.',
  },
  smsp__pcsamp_warps_issue_stalled_warpgroup_arrive_not_issued: {
    reasonLabel: 'Warpgroup Arrive (Not Issued)',
    description: 'Warp stalled waiting on a WARPGROUP.ARRIVES or WARPGROUP.WAIT instruction.',
  },
};

// Discriminated union returned by the unified resolver.
export type GpuFrameInfo =
  | {kind: 'sass'; entry: SASSEntry; sourceLabel: string; sourceUrl: string}
  | {kind: 'stall'; entry: StallEntry; sourceLabel: string; sourceUrl: string};

// Build a deep link to the specific stall reason on the Nsight Compute
// page, using a URL text fragment (`:~:text=…`) so the browser scrolls to
// and highlights the matching prose.
function stallSourceUrl(name: string): string {
  return `${STALL_SOURCE_URL}:~:text=${encodeURIComponent(name)}`;
}

// Resolves a frame name to its GPU info, or undefined if not a known SASS
// mnemonic or PC-sampling reason. Both tables are exact-match lookups.
export function gpuFrameInfo(name: string): GpuFrameInfo | undefined {
  if (name === '') return undefined;
  const sass = SASS_INSTRUCTION_DESCRIPTIONS[name];
  if (sass !== undefined) {
    return {
      kind: 'sass',
      entry: sass,
      sourceLabel: NVIDIA_DOCS_LABEL,
      sourceUrl: sass.sourceUrl ?? SASS_SOURCE_URL,
    };
  }
  const stall = STALL_REASON_DESCRIPTIONS[name];
  if (stall !== undefined) {
    return {
      kind: 'stall',
      entry: stall,
      sourceLabel: NVIDIA_DOCS_LABEL,
      sourceUrl: stall.sourceUrl ?? stallSourceUrl(name),
    };
  }
  return undefined;
}
