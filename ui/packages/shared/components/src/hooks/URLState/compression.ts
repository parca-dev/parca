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

import * as LZString from 'lz-string';

/**
 * Compress a string value using LZ-String compression
 *
 * @param value - The string to compress
 * @returns Compressed string (Base64-like encoding)
 */
export const compressParam = (value: string): string => {
  const startTime = performance.now();

  try {
    if (value === '' || value.length === 0) {
      return value;
    }

    // Compress using LZ-String Base64 (returns URL-safe Base64 string)
    const compressed = LZString.compressToBase64(value);

    // Only use compression if it actually reduces the size
    if (compressed.length < value.length) {
      // Prefix with 'c:' to indicate this is compressed
      const result = `c:${compressed}`;
      const totalTime = performance.now() - startTime;
      console.log(`[compressParam] ${totalTime.toFixed(2)}ms`);
      return result;
    }

    // If compression doesn't help, return original
    return value;
  } catch (error) {
    const totalTime = performance.now() - startTime;
    console.error(`[compressParam] Error after ${totalTime.toFixed(2)}ms:`, error);
    // If compression fails, return original value
    return value;
  }
};

/**
 * Decompress a LZ-String compressed string
 *
 * @param value - The compressed string (with 'c:' prefix) or uncompressed string
 * @returns Decompressed string
 */
export const decompressParam = (value: string): string => {
  const startTime = performance.now();

  try {
    if (value === '' || value.length === 0) {
      return value;
    }

    // Check if this is a compressed value (has 'c:' prefix)
    if (!value.startsWith('c:')) {
      // Not compressed, return as-is
      return value;
    }

    // Remove the 'c:' prefix
    const compressed = value.substring(2);

    // Decompress using LZ-String Base64
    const decompressed = LZString.decompressFromBase64(compressed);

    if (decompressed === null || decompressed === undefined) {
      throw new Error('Decompression returned null');
    }

    const totalTime = performance.now() - startTime;
    console.log(`[decompressParam] ${totalTime.toFixed(2)}ms`);
    return decompressed;
  } catch (error) {
    const totalTime = performance.now() - startTime;
    console.error(`[decompressParam] Error after ${totalTime.toFixed(2)}ms:`, error);
    // If decompression fails, try to return the value without the prefix
    // This provides backward compatibility
    if (value.startsWith('c:')) {
      return value.substring(2);
    }
    return value;
  }
};

/**
 * Check if a value is compressed (has the compression prefix)
 */
export const isCompressed = (value: string): boolean => {
  return value?.startsWith('c:') ?? false;
};
