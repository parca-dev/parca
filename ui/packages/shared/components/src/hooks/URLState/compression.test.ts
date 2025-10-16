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

import {compressParam, decompressParam, isCompressed} from './compression';

describe('URL Parameter Compression', () => {
  describe('compressParam and decompressParam', () => {
    it('should compress and decompress a simple string', () => {
      const original = 'hello world';
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
    });

    it('should compress and decompress a JSON string', () => {
      const original = JSON.stringify({
        foo: 'bar',
        nested: {a: 1, b: 2, c: 3},
        array: [1, 2, 3, 4, 5],
      });
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
      expect(JSON.parse(decompressed)).toEqual(JSON.parse(original));
    });

    it('should compress and decompress a large string', () => {
      const original = 'a'.repeat(1000);
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
      expect(compressed.length).toBeLessThan(original.length);
    });

    it('should handle empty strings', () => {
      const original = '';
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
    });

    it('should add compression prefix when value is compressed', () => {
      const original = 'a'.repeat(1000);
      const compressed = compressParam(original);

      expect(compressed.startsWith('c:')).toBe(true);
    });

    it('should handle very short strings', () => {
      const original = 'short';
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      // LZ-String can compress even short strings
      // Just verify roundtrip works
      expect(decompressed).toBe(original);
    });

    it('should handle backward compatibility with non-compressed values', () => {
      const original = 'uncompressed-value';
      const decompressed = decompressParam(original);

      expect(decompressed).toBe(original);
    });
  });

  describe('isCompressed', () => {
    it('should return true for compressed values', () => {
      const original = 'a'.repeat(1000);
      const compressed = compressParam(original);

      expect(isCompressed(compressed)).toBe(true);
    });

    it('should return false for non-compressed values', () => {
      const original = 'short';

      expect(isCompressed(original)).toBe(false);
    });
  });

  describe('Compression encoding', () => {
    it('should compress and decompress correctly regardless of encoding', () => {
      // Use a large string to ensure compression happens
      const original = JSON.stringify({
        special: '+=/',
        unicode: '你好世界',
        symbols: '!@#$%^&*()',
        repeated: 'a'.repeat(100), // Make it large enough to compress
      });
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      // Verify roundtrip works correctly
      expect(compressed.startsWith('c:')).toBe(true);
      expect(decompressed).toBe(original);
      expect(compressed.length).toBeLessThan(original.length);
    });

    it('should produce URL-safe output that can be encoded', () => {
      // Use a large string to ensure compression happens
      const original = JSON.stringify({
        filters: ['filter1', 'filter2', 'filter3'],
        groupBy: ['label1', 'label2'],
        query: 'sum(rate(container_cpu_usage_seconds_total[5m]))',
        repeated: 'x'.repeat(100),
      });

      const compressed = compressParam(original);

      // Verify it's actually compressed
      expect(compressed.startsWith('c:')).toBe(true);

      // Remove the 'c:' prefix to get just the compressed data
      const compressedData = compressed.substring(2);

      // This should not throw URIError
      expect(() => {
        const encoded = encodeURIComponent(compressedData);
        // Verify it can also be decoded back
        const decoded = decodeURIComponent(encoded);
        expect(decoded).toBe(compressedData);
      }).not.toThrow();

      // Verify full roundtrip through URL encoding
      const compressedData2 = compressed.substring(2);
      const encoded = encodeURIComponent(compressedData2);
      const decoded = decodeURIComponent(encoded);
      const decompressed = decompressParam(`c:${decoded}`);

      expect(decompressed).toBe(original);
    });
  });

  describe('Real-world scenarios', () => {
    it('should handle complex query expression', () => {
      const original =
        'sum(rate(container_cpu_usage_seconds_total{namespace="default"}[5m])) by (pod)';
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
    });

    it('should handle array of labels', () => {
      const original = 'label1,label2,label3,label4,label5';
      const compressed = compressParam(original);
      const decompressed = decompressParam(compressed);

      expect(decompressed).toBe(original);
    });
  });
});
