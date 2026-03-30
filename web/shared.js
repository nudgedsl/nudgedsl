// nudgeDSL shared.js — parser, utilities, common interactions

// ── TOKEN TYPES ──────────────────────────────────────────────────────────
const TK = {ATOM:'A',STR:'S',INT:'I',FLOAT:'F',BOOL:'B',NULL:'N',LP:'(',RP:')',COMMA:',',CHAIN:'>>',FALLBACK:'|',PARALLEL:'//',AMPLIFY:'**',EOF:'$'};

function tokenize(input) {
  if (!input.trim()) return {err:{code:'EMPTY_INPUT',pos:0,msg:'Input is empty or whitespace only'}};
  const tokens = []; let i = 0;
  while (i < input.length) {
    while (i < input.length && ' \t\n\r'.includes(input[i])) i++;
    if (i >= input.length) break;
    const start = i, ch = input[i];
    if (ch === '"') {
      i++;
      while (i < input.length && input[i] !== '"') i++;
      if (i >= input.length) return {err:{code:'UNTERMINATED_STRING',pos:start,msg:`String at position ${start} was never closed`}};
      tokens.push({k:TK.STR, v:input.slice(start+1,i), pos:start}); i++;
    } else if (ch==='('){tokens.push({k:TK.LP,v:'(',pos:i++});}
    else if (ch===')'){tokens.push({k:TK.RP,v:')',pos:i++});}
    else if (ch===','){tokens.push({k:TK.COMMA,v:',',pos:i++});}
    else if (ch==='>'&&input[i+1]==='>'){tokens.push({k:TK.CHAIN,v:'>>',pos:i});i+=2;}
    else if (ch==='|'){tokens.push({k:TK.FALLBACK,v:'|',pos:i++});}
    else if (ch==='/'&&input[i+1]==='/'){tokens.push({k:TK.PARALLEL,v:'//',pos:i});i+=2;}
    else if (ch==='*'&&input[i+1]==='*'){tokens.push({k:TK.AMPLIFY,v:'**',pos:i});i+=2;}
    else if (ch==='-'||(ch>='0'&&ch<='9')) {
      let s=i; if(ch==='-')i++;
      while(i<input.length&&input[i]>='0'&&input[i]<='9')i++;
      if(i<input.length&&input[i]==='.'){i++;while(i<input.length&&input[i]>='0'&&input[i]<='9')i++;tokens.push({k:TK.FLOAT,v:input.slice(s,i),pos:s});}
      else tokens.push({k:TK.INT,v:input.slice(s,i),pos:s});
    } else if (ch>='A'&&ch<='Z') {
      let s=i;
      while(i<input.length&&((input[i]>='A'&&input[i]<='Z')||(input[i]>='0'&&input[i]<='9')))i++;
      tokens.push({k:TK.ATOM,v:input.slice(s,i),pos:s});
    } else if (ch>='a'&&ch<='z') {
      let s=i;
      while(i<input.length&&(input[i]>='a'&&input[i]<='z'))i++;
      const w=input.slice(s,i);
      if(w==='true'||w==='false') tokens.push({k:TK.BOOL,v:w,pos:s});
      else if(w==='null') tokens.push({k:TK.NULL,v:w,pos:s});
      else return {err:{code:'UNKNOWN_ATOM',pos:s,msg:`"${w}" is not a valid atom — atoms must be UPPERCASE`}};
    } else if (ch==='#') {
      return {err:{code:'UNEXPECTED_TOKEN',pos:i,msg:`"#" is not valid — nudgeDSL has no comment syntax`}};
    } else if (ch==='`') {
      return {err:{code:'UNEXPECTED_TOKEN',pos:i,msg:`Markdown backticks detected — output plain nudgeDSL only`}};
    } else {
      return {err:{code:'UNEXPECTED_TOKEN',pos:i,msg:`Unexpected character "${ch}" at position ${i}`}};
    }
  }
  tokens.push({k:TK.EOF,v:'',pos:i});
  return {tokens};
}

function parse(tokens) {
  let pos = 0;
  const cur = () => tokens[pos]||{k:TK.EOF};
  const adv = () => pos++;
  const isOp = k => [TK.CHAIN,TK.FALLBACK,TK.PARALLEL,TK.AMPLIFY].includes(k);

  function expr() { return fallback(); }
  function fallback() {
    let l=parallel(); if(l.err) return l;
    if(cur().k!==TK.FALLBACK) return l;
    const nodes=[l];
    while(cur().k===TK.FALLBACK){adv();if(cur().k===TK.EOF||isOp(cur().k))return{err:{code:'TRAILING_OPERATOR',pos:cur().pos,msg:'| has no right-hand side'}};const r=parallel();if(r.err)return r;nodes.push(r);}
    return {type:'fallback',nodes};
  }
  function parallel() {
    let l=chain(); if(l.err) return l;
    if(cur().k!==TK.PARALLEL) return l;
    const nodes=[l];
    while(cur().k===TK.PARALLEL){adv();if(cur().k===TK.EOF||isOp(cur().k))return{err:{code:'TRAILING_OPERATOR',pos:cur().pos,msg:'// has no right-hand side'}};const r=chain();if(r.err)return r;nodes.push(r);}
    return {type:'parallel',nodes,failureMode:'fail-fast'};
  }
  function chain() {
    let l=amplify(); if(l.err) return l;
    if(cur().k!==TK.CHAIN) return l;
    const nodes=[l];
    while(cur().k===TK.CHAIN){adv();if(cur().k===TK.EOF||isOp(cur().k))return{err:{code:'TRAILING_OPERATOR',pos:cur().pos,msg:'>> has no right-hand side'}};const r=amplify();if(r.err)return r;nodes.push(r);}
    return {type:'chain',nodes};
  }
  function amplify() {
    let n=primary(); if(n.err) return n;
    if(cur().k!==TK.AMPLIFY) return n;
    adv();
    if(cur().k!==TK.INT) return{err:{code:'UNEXPECTED_TOKEN',pos:cur().pos,msg:'** must be followed by a positive integer'}};
    const c=parseInt(cur().v); if(c<1)return{err:{code:'UNEXPECTED_TOKEN',pos:cur().pos,msg:'Amplify count must be >= 1'}};
    adv(); return {type:'amplify',node:n,count:c};
  }
  function primary() {
    const t=cur();
    if(t.k===TK.ATOM) return call();
    if(t.k===TK.LP){adv();if(cur().k===TK.RP)return{err:{code:'UNEXPECTED_TOKEN',pos:cur().pos,msg:'Empty grouping () is not valid'}};const i=expr();if(i.err)return i;if(cur().k!==TK.RP)return{err:{code:'MISSING_CLOSE_PAREN',pos:cur().pos,msg:`Expected ) to close group, got "${cur().v}"`}};adv();return i;}
    if(t.k===TK.EOF) return{err:{code:'TRUNCATED_INPUT',pos:t.pos,msg:'Expression ended unexpectedly — input may be truncated'}};
    if(isOp(t.k)) return{err:{code:'UNEXPECTED_TOKEN',pos:t.pos,msg:`Operator "${t.v}" at start of expression`}};
    return{err:{code:'UNEXPECTED_TOKEN',pos:t.pos,msg:`Unexpected token "${t.v}"`}};
  }
  function call() {
    const a=cur(); adv();
    if(cur().k!==TK.LP) return{err:{code:'UNEXPECTED_TOKEN',pos:cur().pos,msg:`Expected ( after ${a.v}, got "${cur().v}"`}};
    adv();
    const args=[];
    if(cur().k!==TK.RP){
      while(true){const arg=parseArg();if(arg.err)return arg;args.push(arg);if(cur().k!==TK.COMMA)break;adv();}
    }
    if(cur().k===TK.EOF) return{err:{code:'TRUNCATED_INPUT',pos:cur().pos,msg:`${a.v}() call truncated — missing )`}};
    if(cur().k!==TK.RP) return{err:{code:'MISSING_CLOSE_PAREN',pos:cur().pos,msg:`Expected ) to close ${a.v}(), got "${cur().v}"`}};
    adv(); return {type:'call',atom:a.v,args};
  }
  function parseArg() {
    const t=cur();
    if(t.k===TK.STR){adv();return{type:'string',value:t.v};}
    if(t.k===TK.INT){adv();return{type:'integer',value:parseInt(t.v)};}
    if(t.k===TK.FLOAT){adv();return{type:'float',value:parseFloat(t.v)};}
    if(t.k===TK.BOOL){adv();return{type:'boolean',value:t.v==='true'};}
    if(t.k===TK.NULL){adv();return{type:'null',value:null};}
    if(t.k===TK.EOF) return{err:{code:'TRUNCATED_INPUT',pos:t.pos,msg:'Argument list truncated at EOF'}};
    return{err:{code:'UNEXPECTED_TOKEN',pos:t.pos,msg:`Expected argument, got "${t.v}"`}};
  }
  const root=expr();
  if(root.err) return root;
  if(cur().k!==TK.EOF) return{err:{code:'UNEXPECTED_TOKEN',pos:cur().pos,msg:`Unexpected token after expression: "${cur().v}"`}};
  return {version:'0.1.0',root};
}

// ── PUBLIC PARSE API ─────────────────────────────────────────────────────
function nudgeParse(input) {
  const lex = tokenize(input);
  if (lex.err) return {error: lex.err, ast: null};
  const ast = parse(lex.tokens);
  if (ast.err) return {error: ast.err, ast: null};
  return {error: null, ast};
}

// ── TOKEN ESTIMATOR ──────────────────────────────────────────────────────
function estimateTokens(text) {
  return Math.max(1, Math.ceil(text.length / 4));
}

// ── COPY UTILITY ─────────────────────────────────────────────────────────
function copyText(text, btn, label = 'Copy') {
  navigator.clipboard.writeText(text).then(() => {
    const orig = btn.textContent;
    btn.textContent = '✓ Copied';
    btn.classList.add('copied');
    setTimeout(() => { btn.textContent = orig; btn.classList.remove('copied'); }, 2000);
  });
}

// ── TAB SWITCHING ─────────────────────────────────────────────────────────
function switchTab(panelId, btn) {
  const container = btn.closest('section') || document;
  container.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
  container.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  const panel = document.getElementById(panelId);
  if (panel) panel.classList.add('active');
  btn.classList.add('active');
}

// ── NUDGE RULES ───────────────────────────────────────────────────────────
const NUDGE_RULES = `# Nudge — AI Session Framework
I follow the Nudge methodology. Read this before we start.

## 7 Nudge Rules
1. ONE TASK PER CONVERSATION. We do not carry dead history into new phases.
2. Read the HANDOVER. Do not ask me to repeat decisions already made.
3. BRIEF BEFORE EXECUTION. Never execute without a shard.
4. REVIEW BEFORE DONE. Check against the shard.
5. TWO ATTEMPTS MAX. If you miss twice, the shard needs rewriting.
6. ONLY WHAT'S NEEDED. I will supply the relevant context. Do not guess.
7. SOURCE OF TRUTH. context.md and index.md are locked.
---
`;

// ── TRANSLATOR (Anthropic API) ───────────────────────────────────────────
async function translateWithClaude(prose, apiKey, onLoading, onResult, onError) {
  onLoading();
  const inputTokens = estimateTokens(prose);
  try {
    const res = await fetch('https://api.anthropic.com/v1/messages', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': apiKey,
        'anthropic-version': '2023-06-01',
        'anthropic-dangerous-direct-browser-access': 'true'
      },
      body: JSON.stringify({
        model: 'claude-sonnet-4-20250514',
        max_tokens: 1024,
        system: `You are a nudgeDSL v0.1.0 translator. Convert any input (JSON, prose, YAML, handover notes, shard documents) into valid nudgeDSL.

Grammar:
- ATOM(args) — uppercase 1-3 char atom name, args in parens
- >> chain (sequential), | fallback, // parallel, **N amplify
- Args: "string", integer, float, boolean (true/false), null

Common atoms: MARK(id,status), CREATE(path), MODIFY(path), NOTE(text), FETCH(source), NOTIFY(channel), TEST(name), ACCEPT(criterion), FLAG(type,scope), SHARD(id,phase), PHASE(name,desc), VERIFY(id), RESULT(verdict)

Rules:
- Output ONLY valid nudgeDSL — no explanation, no preamble, no markdown fences
- Use >> for sequential actions
- Use // for actions that happen simultaneously  
- Use | for fallback alternatives
- Keep it dense — that's the point`,
        messages: [{role:'user', content:`Translate this to nudgeDSL:\n\n${prose}`}]
      })
    });
    const data = await res.json();
    if (data.error) { onError(data.error.message); return; }
    const dsl = data.content[0].text.trim();
    const outputTokens = estimateTokens(dsl);
    const promptTokens = 115;
    const gross = inputTokens - outputTokens;
    const net = gross - Math.round(promptTokens / 10); // amortized over 10 calls
    const pct = Math.round((net / inputTokens) * 100);
    onResult({dsl, inputTokens, outputTokens, promptTokens, net, pct});
  } catch(e) {
    onError(e.message);
  }
}
