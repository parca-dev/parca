# URLState Hook Usage Guide

The `useURLState` hook provides a simple way to sync component state with URL query parameters. It now includes built-in batching support for efficient URL updates.

## Basic Usage

```tsx
import { useURLState } from '@parca/components';

function MyComponent() {
  const [colorBy, setColorBy] = useURLState('color_by', {
    defaultValue: 'function'
  });

  const [groupBy, setGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: ['function_name'],
    alwaysReturnArray: true
  });

  // Use the state values and setters as normal
  return (
    <div>
      <button onClick={() => setColorBy('filename')}>
        Change Color
      </button>
    </div>
  );
}
```

## Batching Multiple Updates

When you need to update multiple URL parameters simultaneously, use `useURLStateBatch` to ensure a single URL update:

```tsx
import { useURLState, useURLStateBatch } from '@parca/components';

function ProfileFilters() {
  const [colorBy, setColorBy] = useURLState('color_by');
  const [groupBy, setGroupBy] = useURLState<string[]>('group_by', {
    alwaysReturnArray: true
  });
  const [view, setView] = useURLState('view');

  // Get the batch function
  const batchUpdates = useURLStateBatch();

  const handleComplexFilterChange = () => {
    // Batch multiple URL updates into a single navigation
    batchUpdates(() => {
      setColorBy('filename');
      setGroupBy(['function_name', 'filename']);
      setView('table');
    });
    // Results in ONE URL update instead of three!
  };

  return (
    <button onClick={handleComplexFilterChange}>
      Apply All Filters
    </button>
  );
}
```

## Key Features

### Automatic URL Synchronization
- All URL updates are now handled centrally by the `URLStateProvider`
- Individual hooks only manage state; URL sync happens automatically
- Built-in debouncing prevents excessive URL updates

### Batching Support
- Use `batchUpdates` to group multiple parameter changes
- Prevents multiple browser history entries
- Improves performance for complex state updates
- Essential for maintaining URL coherence when multiple related parameters change

## Migration from Direct Navigation

Previously, the ProfileSelector component managed URL updates directly:

With the new approach, you can use individual `useURLState` hooks with batching:

```tsx
// New approach - automatic URL sync with batching
const [expression, setExpression] = useURLState('expression_a');
const [from, setFrom] = useURLState('from_a');
const [to, setTo] = useURLState('to_a');
const batchUpdates = useURLStateBatch();

const selectQuery = (q: QuerySelection): void => {
  batchUpdates(() => {
    setExpression(q.expression);
    setFrom(q.from.toString());
    setTo(q.to.toString());
    // All updates result in a single URL change
  });
};
```

## Benefits

1. **Simpler Code**: No need to manually construct URL parameter objects
2. **Better Performance**: Batching prevents multiple rapid URL updates
3. **Cleaner History**: One history entry instead of multiple for related changes
4. **Type Safety**: Each parameter is individually typed
5. **Easier Testing**: URL synchronization logic is centralized and testable
