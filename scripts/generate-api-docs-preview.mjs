import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(__dirname, '..')
const sourcePath = path.join(repoRoot, 'backend/internal/service/docs/api_reference.md')
const outputPath = path.join(repoRoot, 'backend/internal/service/docs/api_reference.html')

const pageOrder = ['common', 'openai', 'anthropic', 'gemini', 'grok', 'antigravity', 'vertex-batch']
const pageMeta = {
  common: {
    title: '通用接入',
    shortTitle: '通用',
    description: '统一说明基础地址、认证优先级、错误格式、限流语义与文档同步规则。',
  },
  openai: {
    title: 'OpenAI 兼容',
    shortTitle: 'OpenAI',
    description: '聚焦 Responses、Chat Completions、图像与视频入口，以及兼容别名路径。',
  },
  anthropic: {
    title: 'Anthropic / Claude',
    shortTitle: 'Claude',
    description: '说明 Messages、count_tokens、保留头透传，以及 Claude 风格客户端的接入约束。',
  },
  gemini: {
    title: 'Gemini 原生',
    shortTitle: 'Gemini',
    description: '集中展示 models、files、upload/download、batches、live 与 OpenAI compat。',
  },
  grok: {
    title: 'Grok',
    shortTitle: 'Grok',
    description: '整理聊天、responses、图像、视频等 Grok 专用或仅在 Grok 平台生效的能力。',
  },
  antigravity: {
    title: 'Antigravity',
    shortTitle: 'AG',
    description: '解释 Antigravity 前缀下的 Anthropic 风格入口、Gemini 风格入口与能力边界。',
  },
  'vertex-batch': {
    title: 'Vertex / Batch',
    shortTitle: 'Vertex',
    description: '汇总 Vertex Batch Prediction Jobs 与 Google Batch Archive 的特殊调用方式。',
  },
}

const themes = {
  common: ['#0284c7', 'rgba(2,132,199,.12)'],
  openai: ['#059669', 'rgba(5,150,105,.12)'],
  anthropic: ['#d97706', 'rgba(217,119,6,.12)'],
  gemini: ['#2563eb', 'rgba(37,99,235,.12)'],
  grok: ['#e11d48', 'rgba(225,29,72,.12)'],
  antigravity: ['#0f766e', 'rgba(15,118,110,.12)'],
  'vertex-batch': ['#475569', 'rgba(71,85,105,.12)'],
}

const previewIconBase = '../../web/dist/lobehub-icons-static-svg/icons'
const pageIcons = {
  common: {
    inline:
      '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 0 0 6 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 0 1 6 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 0 1 6-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0 0 18 18a8.967 8.967 0 0 0-6 2.292m0-14.25v14.25"/></svg>'
  },
  openai: { src: `${previewIconBase}/openai.svg`, alt: 'OpenAI' },
  anthropic: { src: `${previewIconBase}/anthropic.svg`, alt: 'Anthropic' },
  gemini: { src: `${previewIconBase}/google-color.svg`, alt: 'Gemini' },
  grok: { src: `${previewIconBase}/grok.svg`, alt: 'Grok' },
  antigravity: { src: `${previewIconBase}/antigravity-color.svg`, alt: 'Antigravity' },
  'vertex-batch': { src: `${previewIconBase}/vertexai-color.svg`, alt: 'Vertex / Batch' },
}

function embed(value) {
  return JSON.stringify(value).replace(/</g, '\\u003C')
}

function html(markdown) {
  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>API 文档预览</title>
  <style>
    :root{--accent:#0284c7;--accent-soft:rgba(2,132,199,.12);--bg:#f4f7fb;--text:#111111;--subtext:#1f2937;--border:rgba(148,163,184,.25);--surface:rgba(255,255,255,.92);--shadow:0 24px 60px rgba(15,23,42,.08)}
    *{box-sizing:border-box}
    html{scroll-behavior:smooth}
    body{margin:0;font-family:"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:radial-gradient(circle at top left,rgba(59,130,246,.08),transparent 26%),var(--bg);color:var(--text)}
    a{text-decoration:none;color:inherit}
    button{font:inherit}
    .shell{max-width:1540px;margin:0 auto;padding:24px}
    .hero,.card,.panel{border:1px solid var(--border);background:var(--surface);box-shadow:var(--shadow);backdrop-filter:blur(14px)}
    .hero{border-radius:32px;padding:28px;background:linear-gradient(135deg,rgba(255,255,255,.96),rgba(241,245,249,.88))}
    .hero-top{display:flex;justify-content:space-between;gap:18px;align-items:end;flex-wrap:wrap}
    .eyebrow{font-size:12px;letter-spacing:.28em;text-transform:uppercase;font-weight:700;color:var(--accent)}
    h1{margin:14px 0 0;font-size:clamp(30px,4vw,54px);line-height:1.02;letter-spacing:-.03em;color:var(--text)}
    .hero p{margin:14px 0 0;max-width:860px;color:var(--subtext);line-height:1.85}
    .hero code,.pill code{color:var(--text);background:rgb(241 245 249);padding:.15em .4em;border-radius:8px}
    .actions{display:flex;gap:12px;flex-wrap:wrap}
    .btn{border:none;border-radius:999px;padding:12px 18px;font-weight:700;cursor:pointer}
    .btn.primary{background:var(--accent);color:#fff}
    .btn.secondary{background:#fff;border:1px solid var(--border);color:var(--text)}
    .summary{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px;margin-top:20px}
    .summary .item{border:1px solid var(--border);border-radius:22px;padding:18px;background:rgba(255,255,255,.82)}
    .summary .k{font-size:11px;letter-spacing:.24em;text-transform:uppercase;color:var(--subtext);font-weight:700}
    .summary .v{margin-top:12px;font-size:30px;font-weight:700;line-height:1;color:var(--text)}
    .summary .d{margin-top:10px;font-size:13px;line-height:1.7;color:var(--subtext)}
    .mobile{display:none;gap:12px;flex-direction:column;margin-top:20px}
    .scrollchips{display:flex;gap:10px;overflow:auto;padding-bottom:4px}
    .chip{display:inline-flex;align-items:center;gap:8px;white-space:nowrap;border-radius:999px;border:1px solid var(--border);padding:10px 14px;background:#fff;color:var(--text);font-size:14px;font-weight:500}
    .chip.active{background:var(--accent-soft);border-color:transparent;color:var(--accent)}
    .grid{display:grid;grid-template-columns:280px minmax(0,1fr) 220px;gap:18px;margin-top:26px}
    .sticky{position:sticky;top:0;align-self:start;max-height:100vh;overflow:auto;padding-right:4px}
    .panel{border-radius:28px;padding:14px}
    .ptitle{margin:2px 6px 12px;font-size:11px;letter-spacing:.24em;text-transform:uppercase;color:var(--subtext);font-weight:700}
    .nav,.toc{display:flex;flex-direction:column;gap:10px}
    .nav a,.toc a{display:block;border:1px solid transparent;border-radius:20px;padding:14px;transition:.18s}
    .nav a:hover,.toc a:hover{border-color:var(--border);background:#f8fafc}
    .nav a.active,.toc a.active{background:var(--accent-soft);color:var(--accent)}
    .nav-card{display:flex;align-items:flex-start;gap:12px}
    .nav-icon-shell{display:inline-flex;align-items:center;justify-content:center;width:46px;height:46px;flex-shrink:0;border-radius:18px;border:1px solid rgba(148,163,184,.22);background:rgba(255,255,255,.96);box-shadow:0 12px 28px rgba(15,23,42,.08);color:var(--text)}
    .nav-icon-shell--chip{width:24px;height:24px;border-radius:999px;box-shadow:none}
    .nav-icon-shell svg{width:22px;height:22px}
    .nav-icon-img{width:22px;height:22px;object-fit:contain}
    .nav-copy{min-width:0;flex:1}
    .nav-copy strong{display:block;font-size:14px;color:var(--text)}
    .nav-copy p{margin:6px 0 0;color:var(--subtext);font-size:12px;line-height:1.6}
    .chip-label{display:inline-block}
    .toc a{font-size:13px;color:var(--text);font-weight:500;padding:12px 14px}
    .card{border-radius:32px;overflow:hidden;background:#fff}
    .cardhead{padding:28px;border-bottom:1px solid var(--border);background:linear-gradient(135deg,rgba(255,255,255,.98),rgba(248,250,252,.88)),radial-gradient(circle at top right,var(--accent-soft),transparent 52%)}
    .badge{display:inline-flex;border-radius:999px;background:var(--accent-soft);color:var(--accent);padding:8px 12px;font-size:11px;letter-spacing:.24em;text-transform:uppercase;font-weight:700}
    .card h2{margin:16px 0 0;font-size:clamp(28px,3vw,40px);line-height:1.05;letter-spacing:-.03em;color:var(--text)}
    .cardhead p{margin:14px 0 0;color:var(--subtext);line-height:1.8}
    .body{padding:26px 28px 34px}
    .markdown h1,.markdown h2,.markdown h3,.markdown h4{line-height:1.2;letter-spacing:-.02em;color:var(--text)}
    .markdown p,.markdown li,.markdown blockquote,.markdown td,.markdown th,.markdown strong,.markdown em{color:var(--subtext);line-height:1.9;font-size:15px}
    .markdown ul,.markdown ol{padding-left:1.4rem}
    .markdown li+li{margin-top:8px}
    .markdown a{color:var(--text);text-decoration:underline;text-underline-offset:.2em}
    .markdown blockquote{border-left:4px solid rgb(148 163 184);padding:12px 14px;background:rgb(248 250 252);border-radius:0 16px 16px 0}
    .markdown code{background:rgb(241 245 249);border-radius:8px;padding:.18em .45em;font-size:.92em;color:var(--text)}
    .markdown pre{margin:0;overflow:auto;padding:16px 18px;border-radius:16px;border:1px solid var(--border);background:rgb(248 250 252)}
    .markdown pre code{display:block;color:var(--text);background:transparent;padding:0;white-space:pre}
    .markdown table{width:100%;border-collapse:collapse;border:1px solid var(--border);border-radius:18px;overflow:hidden;display:block}
    .markdown th,.markdown td{padding:12px 14px;border-bottom:1px solid var(--border);text-align:left;vertical-align:top}
    .markdown th{background:rgba(148,163,184,.08);font-weight:700}
    .markdown tr:last-child td{border-bottom:none}
    .sec{scroll-margin-top:32px}
    .sec+.sec{margin-top:34px}
    .sec h3{margin:0 0 16px;font-size:24px;line-height:1.2;color:var(--text)}
    .code{border-radius:24px;overflow:hidden;background:linear-gradient(180deg,#0f172a,#020617);box-shadow:0 18px 40px rgba(15,23,42,.16)}
    .tabs{display:flex;gap:10px;flex-wrap:wrap;padding:16px;border-bottom:1px solid rgba(255,255,255,.08)}
    .tab{border:none;border-radius:999px;padding:9px 14px;background:rgba(255,255,255,.1);color:#f8fafc;font-size:12px;font-weight:700;cursor:pointer}
    .tab.active{background:var(--accent);color:#fff}
    .lang{padding:0 18px;color:#cbd5e1;line-height:40px;font-size:12px;border-bottom:1px solid rgba(255,255,255,.08)}
    .code pre{margin:0;overflow:auto;padding:20px 22px 22px}
    .code pre code{display:block;color:#f8fafc;font-family:"JetBrains Mono","Cascadia Code",Consolas,monospace;font-size:13px;line-height:1.8;white-space:pre}
    .foot{display:flex;gap:12px;flex-wrap:wrap;justify-content:space-between;margin-top:18px;color:var(--subtext);font-size:13px}
    .pill{border:1px solid var(--border);border-radius:999px;padding:9px 12px;background:#fff}
    @media (max-width:1320px){.shell{padding:20px}.grid{grid-template-columns:240px minmax(0,1fr) 200px;gap:16px}}
    @media (max-width:1024px){.grid{grid-template-columns:200px minmax(0,1fr) 180px;gap:14px}.panel{padding:12px}}
    @media (max-width:899px){.mobile{display:flex}.grid{grid-template-columns:1fr}.leftcol,.rightcol{display:none}.body,.cardhead{padding-left:20px;padding-right:20px}}
    @media (max-width:960px){.summary{grid-template-columns:1fr}}
    @media (max-width:640px){.shell{padding:14px}.hero{padding:22px}.hero p,.markdown p,.markdown li{font-size:14px}.sec h3{font-size:21px}}
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <div class="hero-top">
        <div>
          <div class="eyebrow">API Docs Preview</div>
          <h1>站内 API 文档页面预览</h1>
          <p>这是从 <code>backend/internal/service/docs/api_reference.md</code> 派生的静态预览页，用来直观看当前三栏协议文档站的大致呈现效果。左侧切协议，中间看正文，右侧目录会随着滚动高亮；在桌面宽度足够时，左右侧栏会跟随滚动持续停留在视口内，便于深度阅读。</p>
        </div>
        <div class="actions">
          <button id="copyBtn" class="btn primary" type="button">复制全部 Markdown</button>
          <button id="pathBtn" class="btn secondary" type="button">显示源文件路径</button>
        </div>
      </div>
      <div class="summary" id="summary"></div>
    </section>
    <div class="mobile">
      <div class="panel"><div class="ptitle">支持协议</div><div class="scrollchips" id="mobileNav"></div></div>
      <div class="panel" id="mobileTocPanel" hidden><div class="ptitle">本页目录</div><div class="scrollchips" id="mobileToc"></div></div>
    </div>
    <div class="grid">
      <aside class="leftcol"><div class="sticky"><nav class="panel"><div class="ptitle">支持协议</div><div class="nav" id="nav"></div></nav></div></aside>
      <section>
        <div id="article"></div>
        <div class="foot">
          <span class="pill">预览文件：<code>backend/internal/service/docs/api_reference.html</code></span>
          <span class="pill">基线文件：<code>backend/internal/service/docs/api_reference.md</code></span>
        </div>
      </section>
      <aside class="rightcol"><div class="sticky"><nav class="panel" id="tocPanel" hidden><div class="ptitle">本页目录</div><div class="toc" id="toc"></div></nav></div></aside>
    </div>
  </main>
  <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/dompurify@3.3.1/dist/purify.min.js"></script>
  <script>
    const RAW = ${embed(markdown)};
    const ORDER = ${embed(pageOrder)};
    const META = ${embed(pageMeta)};
    const THEMES = ${embed(themes)};
    const PAGE_ICONS = ${embed(pageIcons)};
    const TAB_ORDER = ['Python', 'Javascript', 'Rest'];
    const EMPTY_PAGE_TEXT = '> 当前协议页尚未写入内容，请在管理页补齐对应章节。';

    marked.setOptions({ gfm: true, breaks: false, headerIds: false, mangle: false });

    const state = { doc: parseDocs(RAW), pageId: getPageId(), activeSectionId: '', observer: null };
    const $ = (selector) => document.querySelector(selector);

    render();
    bind();

    function bind() {
      $('#copyBtn').onclick = async () => {
        try {
          await navigator.clipboard.writeText(RAW);
          alert('Markdown 已复制。');
        } catch {
          alert('复制失败，请手动从源文件复制。');
        }
      };

      $('#pathBtn').onclick = () => {
        alert('源文件：backend/internal/service/docs/api_reference.md');
      };

      window.addEventListener('popstate', () => {
        state.pageId = getPageId();
        render();
      });
    }

    function getPageId() {
      const value = new URL(location.href).searchParams.get('page');
      return ORDER.includes(value) ? value : 'common';
    }

    function getCurrentPage() {
      return state.doc.pages.find((page) => page.id === state.pageId) || state.doc.pages[0];
    }

    function buildPageHref(id) {
      const url = new URL(location.href);
      url.searchParams.set('page', id);
      url.hash = '';
      return url.toString();
    }

    function setTheme(id) {
      document.documentElement.style.setProperty('--accent', THEMES[id][0]);
      document.documentElement.style.setProperty('--accent-soft', THEMES[id][1]);
    }

    function render() {
      const page = getCurrentPage();
      setTheme(page.id);
      renderSummary();
      renderNav();
      renderArticle(page);
      renderToc(page);
      observeSections();
    }

    function renderSummary() {
      $('#summary').innerHTML = [
        ['协议页数', state.doc.pages.length + ' 个', '覆盖通用接入、OpenAI、Claude、Gemini、Grok、Antigravity 与 Vertex / Batch。'],
        ['代码示例', 'Python / Javascript / Rest', '每个协议页都按多语言标签页展示示例。'],
        ['当前域名', 'api.zyxai.de', '预览内容直接来自当前仓库里的 Markdown 基线。'],
      ].map(([key, value, description]) =>
        '<article class="item"><div class="k">' + escapeHtml(key) + '</div><div class="v">' + escapeHtml(value) + '</div><div class="d">' + escapeHtml(description) + '</div></article>'
      ).join('');
    }

    function renderNav() {
      const currentPage = getCurrentPage();

      $('#nav').innerHTML = state.doc.pages.map((page) =>
        '<a href="' + buildPageHref(page.id) + '" data-page="' + page.id + '" class="' + (page.id === currentPage.id ? 'active' : '') + '"><div class="nav-card">' + renderPageIcon(page.id) + '<div class="nav-copy"><strong>' + escapeHtml(page.title) + '</strong><p>' + escapeHtml(page.description) + '</p></div></div></a>'
      ).join('');

      $('#mobileNav').innerHTML = state.doc.pages.map((page) =>
        '<a href="' + buildPageHref(page.id) + '" data-page="' + page.id + '" class="chip ' + (page.id === currentPage.id ? 'active' : '') + '">' + renderPageIcon(page.id, true) + '<span class="chip-label">' + escapeHtml(page.shortTitle) + '</span></a>'
      ).join('');

      document.querySelectorAll('[data-page]').forEach((link) => {
        link.onclick = (event) => {
          event.preventDefault();
          const nextId = link.getAttribute('data-page');
          if (!nextId || nextId === state.pageId) {
            return;
          }

          state.pageId = nextId;
          const url = new URL(location.href);
          url.searchParams.set('page', nextId);
          url.hash = '';
          history.pushState({}, '', url);
          render();
          window.scrollTo({ top: 0, behavior: 'smooth' });
        };
      });
    }

    function renderArticle(page) {
      const introHtml = renderBlocks(page.introBlocks);
      const sectionsHtml = page.sections.map((section) =>
        '<section class="sec" id="' + escapeHtml(section.id) + '" data-docs-section="' + escapeHtml(section.id) + '"><h3>' + escapeHtml(section.title) + '</h3>' + renderBlocks(section.contentBlocks) + '</section>'
      ).join('');

      $('#article').innerHTML =
        '<article class="card"><header class="cardhead"><span class="badge">' + escapeHtml(page.shortTitle) + '</span><h2>' + escapeHtml(page.title) + '</h2><p>' + escapeHtml(page.description) + '</p></header><div class="body">' +
        introHtml +
        (introHtml && sectionsHtml ? '<div style="height:14px"></div>' : '') +
        sectionsHtml +
        '</div></article>';

      document.querySelectorAll('[data-code-group]').forEach((group) => {
        group.querySelectorAll('.tab').forEach((tab) => {
          tab.onclick = () => {
            group.querySelectorAll('.tab').forEach((item) => item.classList.remove('active'));
            tab.classList.add('active');
            group.querySelector('.lang').textContent = tab.getAttribute('data-lang') || '';
            group.querySelector('pre code').textContent = tab.getAttribute('data-code') || '';
          };
        });
      });
    }

    function renderToc(page) {
      if (!page.sections.length) {
        $('#tocPanel').hidden = true;
        $('#mobileTocPanel').hidden = true;
        $('#toc').innerHTML = '';
        $('#mobileToc').innerHTML = '';
        return;
      }

      $('#tocPanel').hidden = false;
      $('#mobileTocPanel').hidden = false;
      state.activeSectionId = page.sections[0].id;

      $('#toc').innerHTML = page.sections.map((section, index) =>
        '<a href="#' + escapeHtml(section.id) + '" data-toc="' + escapeHtml(section.id) + '" class="' + (index === 0 ? 'active' : '') + '">' + escapeHtml(section.title) + '</a>'
      ).join('');

      $('#mobileToc').innerHTML = page.sections.map((section, index) =>
        '<a href="#' + escapeHtml(section.id) + '" data-mobile-toc="' + escapeHtml(section.id) + '" class="chip ' + (index === 0 ? 'active' : '') + '">' + escapeHtml(section.title) + '</a>'
      ).join('');

      document.querySelectorAll('[data-toc],[data-mobile-toc]').forEach((link) => {
        link.onclick = (event) => {
          event.preventDefault();
          const sectionId = link.getAttribute('data-toc') || link.getAttribute('data-mobile-toc');
          document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        };
      });
    }

    function observeSections() {
      if (state.observer) {
        state.observer.disconnect();
        state.observer = null;
      }

      const sections = [...document.querySelectorAll('[data-docs-section]')];
      if (!sections.length || typeof IntersectionObserver === 'undefined') {
        return;
      }

      state.observer = new IntersectionObserver((entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top);
        const nextId = visible[0]?.target?.getAttribute('data-docs-section');
        if (!nextId) {
          return;
        }

        state.activeSectionId = nextId;
        syncTocHighlight();
      }, { rootMargin: '-120px 0px -58% 0px', threshold: [0, 0.2, 1] });

      sections.forEach((section) => state.observer.observe(section));
      syncTocHighlight();
    }

    function syncTocHighlight() {
      document.querySelectorAll('[data-toc]').forEach((link) => {
        link.classList.toggle('active', link.getAttribute('data-toc') === state.activeSectionId);
      });
      document.querySelectorAll('[data-mobile-toc]').forEach((link) => {
        link.classList.toggle('active', link.getAttribute('data-mobile-toc') === state.activeSectionId);
      });
    }

    function renderBlocks(blocks) {
      return blocks.map((block) => {
        if (block.kind === 'markdown') {
          return '<div class="markdown">' + renderMarkdown(block.markdown) + '</div>';
        }
        return renderCodeGroup(block.group);
      }).join('');
    }

    function renderCodeGroup(group) {
      const firstTab = group.tabs[0];
      return '<section class="code" data-code-group="' + escapeHtml(group.id) + '"><div class="tabs">' +
        group.tabs.map((tab, index) =>
          '<button type="button" class="tab ' + (index === 0 ? 'active' : '') + '" data-lang="' + escapeHtml(tab.language) + '" data-code="' + escapeAttribute(tab.code) + '">' + escapeHtml(tab.label) + '</button>'
        ).join('') +
        '</div><div class="lang">' + escapeHtml(firstTab.language) + '</div><pre><code>' + escapeHtml(firstTab.code) + '</code></pre></section>';
    }

    function renderMarkdown(markdown) {
      const raw = marked.parse(String(markdown || '').trim());
      return window.DOMPurify ? DOMPurify.sanitize(raw) : raw;
    }

    function renderPageIcon(pageId, compact = false) {
      const icon = PAGE_ICONS[pageId];
      const shellClass = 'nav-icon-shell' + (compact ? ' nav-icon-shell--chip' : '');
      if (!icon) {
        return '<span class="' + shellClass + '">?</span>';
      }
      if (icon.inline) {
        return '<span class="' + shellClass + '">' + icon.inline + '</span>';
      }
      return '<span class="' + shellClass + '"><img class="nav-icon-img" src="' + escapeAttribute(icon.src) + '" alt="' + escapeAttribute(icon.alt || META[pageId]?.title || pageId) + '" loading="lazy" decoding="async" /></span>';
    }

    function parseDocs(markdown) {
      const lines = normalizeMarkdown(markdown).split('\\n');
      const pages = splitPages(lines);
      return { pages: ORDER.map((id) => buildPage(id, pages.get(id) || [])) };
    }

    function normalizeMarkdown(markdown) {
      return String(markdown || '').replace(/^\\uFEFF/, '').replace(/\\r\\n/g, '\\n');
    }

    function splitPages(lines) {
      const pages = new Map();
      let currentPageId = null;
      let inFence = false;
      let fenceMarker = '';

      for (const line of lines) {
        const fence = parseFence(line);
        if (fence) {
          if (!inFence) {
            inFence = true;
            fenceMarker = fence;
          } else if (fence === fenceMarker) {
            inFence = false;
            fenceMarker = '';
          }
        }

        if (!inFence) {
          const match = line.match(/^##\\s+(.+)$/);
          const pageId = match ? normalizePageId(match[1]) : null;
          if (pageId) {
            currentPageId = pageId;
            if (!pages.has(pageId)) {
              pages.set(pageId, []);
            }
            continue;
          }
        }

        if (currentPageId) {
          pages.get(currentPageId).push(line);
        }
      }

      return pages;
    }

    function buildPage(id, lines) {
      const meta = META[id];
      const sectionsData = splitSections(lines, id);
      return {
        id,
        title: meta.title,
        shortTitle: meta.shortTitle,
        description: meta.description,
        introBlocks: parseBlocks(sectionsData.introLines.length ? sectionsData.introLines : [EMPTY_PAGE_TEXT], 'page-' + id, id),
        sections: sectionsData.sections,
      };
    }

    function splitSections(lines, pageId) {
      const introLines = [];
      const sections = [];
      let collectingIntro = true;
      let currentTitle = '';
      let currentLines = [];
      let inFence = false;
      let fenceMarker = '';
      const counters = new Map();

      const pushSection = () => {
        if (!currentTitle) {
          return;
        }
        sections.push({
          id: createHeadingId(currentTitle, counters),
          title: currentTitle,
          contentBlocks: parseBlocks(currentLines, 'section-' + pageId + '-' + (sections.length + 1), pageId),
        });
      };

      for (const line of lines) {
        const fence = parseFence(line);
        if (fence) {
          if (!inFence) {
            inFence = true;
            fenceMarker = fence;
          } else if (fence === fenceMarker) {
            inFence = false;
            fenceMarker = '';
          }
        }

        if (!inFence) {
          const match = line.match(/^###\\s+(.+)$/);
          if (match) {
            if (collectingIntro) {
              collectingIntro = false;
            } else {
              pushSection();
            }
            currentTitle = normalizeHeadingText(match[1]);
            currentLines = [];
            continue;
          }
        }

        if (collectingIntro) {
          introLines.push(line);
        } else {
          currentLines.push(line);
        }
      }

      pushSection();
      return { introLines, sections };
    }

    function parseBlocks(lines, prefix, pageId) {
      const blocks = [];
      const markdownBuffer = [];
      let cursor = 0;
      let blockIndex = 0;

      const flushMarkdown = () => {
        const markdown = markdownBuffer.join('\\n').trim();
        if (!markdown) {
          markdownBuffer.length = 0;
          return;
        }
        blocks.push({
          id: prefix + '-markdown-' + (blockIndex + 1),
          kind: 'markdown',
          markdown,
        });
        blockIndex += 1;
        markdownBuffer.length = 0;
      };

      while (cursor < lines.length) {
        const codeGroup = parseCodeGroup(lines, cursor, prefix + '-code-' + (blockIndex + 1), pageId);
        if (codeGroup) {
          flushMarkdown();
          blocks.push({
            id: codeGroup.group.id,
            kind: 'code-group',
            group: codeGroup.group,
          });
          blockIndex += 1;
          cursor = codeGroup.nextIndex;
          continue;
        }

        markdownBuffer.push(lines[cursor]);
        cursor += 1;
      }

      flushMarkdown();
      return blocks;
    }

    function parseCodeGroup(lines, startIndex, groupId, pageId) {
      const firstTab = parseCodeTab(lines, startIndex, groupId, 0);
      if (!firstTab) {
        return null;
      }

      const tabs = [firstTab.tab];
      let cursor = firstTab.nextIndex;
      let tabIndex = 1;

      while (true) {
        const nextTab = parseCodeTab(lines, cursor, groupId, tabIndex);
        if (!nextTab) {
          break;
        }
        tabs.push(nextTab.tab);
        cursor = nextTab.nextIndex;
        tabIndex += 1;
      }

      TAB_ORDER.filter((label) => !tabs.some((tab) => tab.label === label)).forEach((label) => {
        tabs.push({
          id: groupId + '-' + label.toLowerCase(),
          label,
          language: defaultLanguage(label),
          code: notApplicableCode(pageId, label),
        });
      });

      tabs.sort((left, right) => TAB_ORDER.indexOf(left.label) - TAB_ORDER.indexOf(right.label));

      return {
        group: {
          id: groupId,
          tabs,
        },
        nextIndex: cursor,
      };
    }

    function parseCodeTab(lines, startIndex, groupId, tabIndex) {
      const heading = lines[startIndex]?.match(/^####\\s+(Python|JavaScript|REST)\\s*$/i);
      if (!heading) {
        return null;
      }

      let cursor = startIndex + 1;
      while (cursor < lines.length && lines[cursor].trim() === '') {
        cursor += 1;
      }

      const fence = parseFence(lines[cursor] || '');
      if (!fence) {
        return null;
      }

      const language = extractFenceInfo(lines[cursor] || '') || defaultLanguage(normalizeTabLabel(heading[1]));
      cursor += 1;

      const codeLines = [];
      while (cursor < lines.length) {
        if (matchesFence(lines[cursor], fence)) {
          cursor += 1;
          break;
        }
        codeLines.push(lines[cursor]);
        cursor += 1;
      }

      while (cursor < lines.length && lines[cursor].trim() === '') {
        cursor += 1;
      }

      return {
        tab: {
          id: groupId + '-tab-' + (tabIndex + 1),
          label: normalizeTabLabel(heading[1]),
          language,
          code: codeLines.join('\\n').replace(/\\n+$/g, ''),
        },
        nextIndex: cursor,
      };
    }

    function normalizePageId(value) {
      const normalized = normalizeHeadingText(value).toLowerCase();
      return ORDER.includes(normalized) ? normalized : null;
    }

    function normalizeTabLabel(value) {
      const normalized = String(value || '').trim().toLowerCase();
      if (normalized === 'javascript') {
        return 'Javascript';
      }
      if (normalized === 'rest') {
        return 'Rest';
      }
      return 'Python';
    }

    function defaultLanguage(label) {
      if (label === 'Javascript') {
        return 'javascript';
      }
      if (label === 'Rest') {
        return 'bash';
      }
      return 'python';
    }

    function notApplicableCode(pageId, label) {
      if (label === 'Javascript') {
        return [
          '// ' + META[pageId].title,
          '// 当前协议暂不提供 ' + label + ' 示例。',
          '// 如需补充，请同步更新仓库中的 api_reference.md 基线文档。',
        ].join('\\n');
      }

      return [
        '# ' + META[pageId].title,
        '# 当前协议暂不提供 ' + label + ' 示例。',
        '# 如需补充，请同步更新仓库中的 api_reference.md 基线文档。',
      ].join('\\n');
    }

    function normalizeHeadingText(value) {
      return String(value || '')
        .replace(/\\[(.*?)\\]\\((.*?)\\)/g, '$1')
        .replace(new RegExp('\`([^\`]+)\`', 'g'), '$1')
        .replace(/\\*\\*(.*?)\\*\\*/g, '$1')
        .replace(/\\*(.*?)\\*/g, '$1')
        .replace(/~~(.*?)~~/g, '$1')
        .replace(/#+$/g, '')
        .trim();
    }

    function createHeadingId(text, counters) {
      const base = slugify(text);
      const count = counters.get(base) || 0;
      counters.set(base, count + 1);
      return count === 0 ? base : base + '-' + (count + 1);
    }

    function slugify(text) {
      const normalized = String(text || '')
        .toLowerCase()
        .trim()
        .replace(/[^\\p{L}\\p{N}\\s-]/gu, '')
        .replace(/\\s+/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '');
      return normalized || 'section';
    }

    function parseFence(line) {
      const match = String(line || '').match(/^\\s*(\`\`\`+|~~~+)/);
      return match ? match[1] : '';
    }

    function matchesFence(line, fence) {
      return new RegExp('^\\\\s*' + escapeRegExp(fence) + '\\\\s*$').test(String(line || ''));
    }

    function extractFenceInfo(line) {
      const match = String(line || '').match(/^\\s*(\`\`\`+|~~~+)\\s*([^\\s]+)?/);
      return (match && match[2] ? match[2].trim() : '');
    }

    function escapeRegExp(value) {
      return String(value || '').replace(/[.*+?^$()|[\\]{}\\\\]/g, '\\\\$&');
    }

    function escapeHtml(value) {
      return String(value || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    }

    function escapeAttribute(value) {
      return escapeHtml(value).replace(/\\n/g, '&#10;');
    }
  </script>
</body>
</html>`;
}

const markdown = (await readFile(sourcePath, 'utf8')).replace(/^\uFEFF/, '')
await mkdir(path.dirname(outputPath), { recursive: true })
await writeFile(outputPath, html(markdown), 'utf8')
console.log(`Generated preview: ${path.relative(repoRoot, outputPath)}`)
