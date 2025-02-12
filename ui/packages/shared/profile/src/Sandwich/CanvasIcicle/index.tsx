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

import {useEffect, useRef, useState} from 'react';

import {type Row as RowType} from '@tanstack/table-core';

import {Row} from '../../Table';

interface Props {
  row: RowType<Row>;
}

interface DrawContext {
  ctx: CanvasRenderingContext2D;
  canvas: HTMLCanvasElement;
  data: Row;
}

const STACK_HEIGHT = 26;
const LABEL_WIDTH = 30;

export const CanvasIcicle = ({row}: Props): JSX.Element => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState({width: 0, height: 400});
  const data = row.original;
  console.log('🚀 ~ data:', row);

  const clearCanvas = ({ctx, canvas}: DrawContext) => {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
  };

  const drawLabels = ({ctx, canvas, data}: DrawContext) => {
    ctx.save();

    ctx.fillStyle = '#ffffff';
    ctx.font = '12px monospace';
    ctx.textAlign = 'center';

    const callers = data.callers || [];
    const callersEndY = callers.length * STACK_HEIGHT;

    // Draw "Callers" label - rotated 90 degrees
    ctx.translate(LABEL_WIDTH / 2, callersEndY); // Move to position
    ctx.rotate(-Math.PI / 2); // Rotate -90 degrees
    ctx.fillText('Callers', 0, 0);
    ctx.restore(); // Restore context state

    // Save context for second label
    ctx.save();

    // Draw "Callees" label - rotated 90 degrees
    const callersHeight = callers.length * (STACK_HEIGHT + 5);
    const currentFunctionY = callersHeight + 10;
    const calleesStartY = currentFunctionY + STACK_HEIGHT + 10; // Position at first callee

    ctx.translate(LABEL_WIDTH / 2, calleesStartY); // Move to position
    ctx.rotate(-Math.PI / 2); // Rotate -90 degrees
    ctx.fillStyle = '#ffffff';
    ctx.font = '12px monospace';
    ctx.fillText('Callees', 0, 0);
    ctx.restore(); // Restore context state
  };

  const drawCallers = ({ctx, canvas, data}: DrawContext) => {
    const callers = data.callers || [];

    callers.forEach((caller, index) => {
      const width = canvas.width - LABEL_WIDTH; // Adjust width to account for labels
      const x = LABEL_WIDTH; // Start after labels
      const y = index * (STACK_HEIGHT + 1);

      // Draw caller block
      ctx.fillStyle = caller.colorProperty?.color || '#c586c0';
      ctx.fillRect(x, y, width, STACK_HEIGHT);

      // Draw caller name
      ctx.fillStyle = '#ffffff';
      ctx.font = '12px monospace';
      ctx.textAlign = 'left';
      ctx.fillText(caller.name || '', x + 10, y + STACK_HEIGHT / 2 + 4);
    });
  };

  const drawCurrentFunction = ({ctx, canvas, data}: DrawContext) => {
    const callers = data.callers || [];
    const callersHeight = callers.length * (STACK_HEIGHT + 5);

    const y = callersHeight + 10;
    const width = canvas.width - LABEL_WIDTH; // Adjust width to account for labels
    const x = LABEL_WIDTH; // Start after labels

    // Draw function block
    ctx.fillStyle = data.colorProperty?.color || '#4ec9b0';
    ctx.fillRect(x, y, width, STACK_HEIGHT);

    // Draw function name
    ctx.fillStyle = '#ffffff';
    ctx.font = '12px monospace';
    ctx.textAlign = 'left';
    ctx.fillText(data.name || '', x + 10, y + STACK_HEIGHT / 2 + 4);
  };

  const drawCallees = ({ctx, canvas, data}: DrawContext) => {
    const callees = data.callees || [];
    const callers = data.callers || [];
    const callersHeight = callers.length * (STACK_HEIGHT + 5);
    const currentFunctionY = callersHeight + 10;

    callees.forEach((callee, index) => {
      const width = canvas.width - LABEL_WIDTH; // Adjust width to account for labels
      const x = LABEL_WIDTH; // Start after labels
      const y = currentFunctionY + STACK_HEIGHT + 10 + index * (STACK_HEIGHT + 1);

      // Draw callee block
      ctx.fillStyle = callee.colorProperty?.color || '#569cd6';
      ctx.fillRect(x, y, width, STACK_HEIGHT);

      // Draw callee name
      ctx.fillStyle = '#ffffff';
      ctx.font = '12px monospace';
      ctx.textAlign = 'left';
      ctx.fillText(callee.name || '', x + 10, y + STACK_HEIGHT / 2 + 4);
    });
  };

  const calculateMinHeight = (data: Row): number => {
    const callers = data.callers || [];
    const callees = data.callees || [];

    const callersHeight = callers.length * (STACK_HEIGHT + 5);
    const calleesHeight = callees.length * (STACK_HEIGHT + 5);
    const currentFunctionHeight = STACK_HEIGHT;
    const spacingHeight = 20; // Reduced from 80 to 20 (10px * 2 for spacing between sections)

    return callersHeight + currentFunctionHeight + calleesHeight + spacingHeight;
  };

  useEffect(() => {
    if (!containerRef.current) return;

    const resizeObserver = new ResizeObserver(entries => {
      for (const entry of entries) {
        const {width, height} = entry.contentRect;
        setDimensions({
          width,
          height,
        });
      }
    });

    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
    };
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || dimensions.width === 0) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const minHeight = calculateMinHeight(data);
    const canvasHeight = Math.max(dimensions.height, minHeight);

    canvas.width = dimensions.width;
    canvas.height = canvasHeight;

    const drawContext = {ctx, canvas, data};

    clearCanvas(drawContext);
    drawLabels(drawContext); // Add labels first
    drawCallers(drawContext);
    drawCurrentFunction(drawContext);
    drawCallees(drawContext);
  }, [data, dimensions]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full overflow-auto"
      style={{
        margin: 0,
        padding: 0,
        height: '100%',
      }}
    >
      <canvas
        ref={canvasRef}
        style={{
          width: '100%',
          minHeight: '100%',
          display: 'block',
          margin: 0,
          padding: 0,
        }}
      />
    </div>
  );
};
