import type { ReactNode } from "react";

// A deliberately small highlighter for the two static samples on the page (Go and HTTP + JSON).
type Rule = [kind: string, pattern: RegExp];

const rules: Rule[] = [
  ["comment", /\/\/[^\n]*/y],
  ["string", /"(?:[^"\\\n]|\\.)*"/y],
  ["placeholder", /<[a-z0-9-]+>/y],
  ["method", /\b(?:GET|POST|PUT|PATCH|DELETE)\b/y],
  ["header", /^[A-Z][A-Za-z-]+(?=:)/my],
  ["keyword", /\b(?:package|import|func|return|var|const|type|if|nil|error|byte)\b/y],
  ["number", /\b\d+\b/y],
  ["call", /\b[A-Za-z_]\w*(?=\()/y],
];

export function highlight(code: string): ReactNode[] {
  const out: ReactNode[] = [];
  let plain = "";
  let i = 0;
  const flush = () => {
    if (plain) out.push(plain);
    plain = "";
  };
  outer: while (i < code.length) {
    for (const [kind, re] of rules) {
      re.lastIndex = i;
      const m = re.exec(code);
      if (m && m[0]) {
        flush();
        out.push(
          <span key={i} className={`tok-${kind}`}>
            {m[0]}
          </span>
        );
        i += m[0].length;
        continue outer;
      }
    }
    // Consume identifiers whole so keywords never match inside longer words.
    const word = /[A-Za-z_]\w*/y;
    word.lastIndex = i;
    const w = word.exec(code);
    const chunk = w ? w[0] : code[i]!;
    plain += chunk;
    i += chunk.length;
  }
  flush();
  return out;
}
