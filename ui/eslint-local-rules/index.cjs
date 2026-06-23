'use strict';

/**
 * Require an explicit `history` option on every *writable* nuqs URL-state hook.
 *
 * nuqs defaults to `history: 'replace'` (no browser history entry). To keep the
 * back button meaningful, user-driven params must opt into `history: 'push'`,
 * while seeding/reset/programmatic writes stay explicit `{history: 'replace'}`.
 * Leaving it implicit silently picks replace and the param drops out of
 * back-button history — this rule makes the choice mandatory so new params
 * can't regress that behaviour by omission.
 *
 * Only fires when a setter is destructured (`const [v, setV] = ...`); read-only
 * declarations don't write the URL, so history is irrelevant there.
 */
const NUQS_HOOKS = new Set(['useQueryState', 'useQueryStates']);

// An options object "specifies history" if it has a `history` key or spreads
// another object (e.g. {...PUSH_HISTORY}). A bare identifier is trusted as a
// named options constant.
function specifiesHistory(node) {
  if (node.type === 'Identifier') return true;
  if (node.type !== 'ObjectExpression') return false;
  return node.properties.some(
    p =>
      p.type === 'SpreadElement' ||
      (p.key != null && (p.key.name === 'history' || p.key.value === 'history'))
  );
}

// Find a `.withOptions(<opts>)` call anywhere in a subtree whose <opts>
// specifies history (covers parser chains, and parsers built inside useMemo).
function chainSpecifiesHistory(node) {
  let found = false;
  const visit = n => {
    if (found || n == null || typeof n.type !== 'string') return;
    if (
      n.type === 'CallExpression' &&
      n.callee.type === 'MemberExpression' &&
      n.callee.property != null &&
      n.callee.property.name === 'withOptions' &&
      n.arguments.length > 0 &&
      specifiesHistory(n.arguments[0])
    ) {
      found = true;
      return;
    }
    for (const key of Object.keys(n)) {
      if (key === 'parent') continue;
      const child = n[key];
      if (Array.isArray(child)) child.forEach(visit);
      else if (child != null && typeof child.type === 'string') visit(child);
    }
  };
  visit(node);
  return found;
}

function isWritable(call) {
  const parent = call.parent;
  if (parent == null || parent.type !== 'VariableDeclarator' || parent.id.type !== 'ArrayPattern') {
    // Not the `const [v, setV] = ...` form — can't prove it's read-only, require it.
    return true;
  }
  return parent.id.elements.length > 1 && parent.id.elements[1] != null;
}

const requireNuqsHistory = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Require an explicit nuqs `history` option on writable URL-state hooks so back-button behaviour is intentional.',
    },
    schema: [],
    messages: {
      missing:
        "nuqs {{name}} needs an explicit `history`: 'push' to record the change in browser history (Back/Forward steps through it), 'replace' to skip it (seeding, resets, programmatic writes).",
    },
  },
  create(context) {
    const localNuqsHooks = new Map(); // localName -> importedName
    const sourceCode = context.sourceCode ?? context.getSourceCode();

    // Resolve an identifier (e.g. a parser stored in a local variable) to its
    // initializer so a memoized/shared parser carrying `.withOptions(...)`
    // isn't a false positive.
    const resolveInit = idNode => {
      let scope = sourceCode.getScope(idNode);
      while (scope != null) {
        const variable = scope.variables.find(v => v.name === idNode.name);
        if (variable != null) {
          const def = variable.defs[0];
          return def != null && def.node.type === 'VariableDeclarator' ? def.node.init : null;
        }
        scope = scope.upper;
      }
      return null;
    };

    const argHasHistory = arg => {
      if (arg.type === 'ObjectExpression' && specifiesHistory(arg)) return true;
      if (chainSpecifiesHistory(arg)) return true;
      if (arg.type === 'Identifier') {
        const init = resolveInit(arg);
        if (init != null) {
          return chainSpecifiesHistory(init) || (init.type === 'ObjectExpression' && specifiesHistory(init));
        }
      }
      return false;
    };

    return {
      ImportDeclaration(node) {
        if (node.source.value !== 'nuqs') return;
        for (const spec of node.specifiers) {
          if (spec.type === 'ImportSpecifier' && NUQS_HOOKS.has(spec.imported.name)) {
            localNuqsHooks.set(spec.local.name, spec.imported.name);
          }
        }
      },
      CallExpression(node) {
        if (node.callee.type !== 'Identifier') return;
        const imported = localNuqsHooks.get(node.callee.name);
        if (imported == null || !isWritable(node)) return;

        // First arg is the key (never carries history); options live after it.
        const hasHistory = node.arguments.slice(1).some(argHasHistory);
        if (!hasHistory) {
          context.report({node, messageId: 'missing', data: {name: imported}});
        }
      },
    };
  },
};

module.exports = {
  'require-nuqs-history': requireNuqsHistory,
};
