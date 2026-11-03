// notesync site behaviour: theme toggle, mobile nav, code highlighting and
// copy buttons, image zoom, table-of-contents tracking, search, live reload.
(function () {
  'use strict';
  var meta = document.querySelector('meta[name="nb-root"]');
  var root = meta ? meta.getAttribute('content') : '';
  var $ = function (s, el) { return (el || document).querySelector(s); };
  var $$ = function (s, el) { return Array.prototype.slice.call((el || document).querySelectorAll(s)); };

  // ---- theme ----
  var themeBtn = $('#theme-toggle');
  if (themeBtn) themeBtn.addEventListener('click', function () {
    var next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = next;
    try { localStorage.setItem('nb-theme', next); } catch (e) {}
  });

  // ---- mobile navigation ----
  var menu = $('#menu-toggle');
  if (menu) menu.addEventListener('click', function () { document.body.classList.toggle('nav-open'); });
  $$('[data-close-nav]').forEach(function (el) {
    el.addEventListener('click', function () { document.body.classList.remove('nav-open'); });
  });
  var active = $('.sidebar a.active');
  if (active && active.scrollIntoView) active.scrollIntoView({ block: 'center' });

  // ---- code blocks ----
  $$('.prose pre > code').forEach(function (code) {
    if (window.hljs) { try { window.hljs.highlightElement(code); } catch (e) {} }
    var btn = document.createElement('button');
    btn.type = 'button'; btn.className = 'copy'; btn.textContent = 'Copy';
    btn.addEventListener('click', function () {
      var done = function (msg) { btn.textContent = msg; setTimeout(function () { btn.textContent = 'Copy'; }, 1400); };
      if (navigator.clipboard) navigator.clipboard.writeText(code.innerText).then(function () { done('Copied'); }, function () { done('Failed'); });
      else done('Failed');
    });
    code.parentElement.appendChild(btn);
  });
  // highlight.js loads with defer; if it arrived after us, highlight again.
  window.addEventListener('load', function () {
    if (!window.hljs) return;
    $$('.prose pre > code:not(.hljs)').forEach(function (c) { try { window.hljs.highlightElement(c); } catch (e) {} });
  });

  // ---- image zoom ----
  var lb = $('#lightbox');
  if (lb) {
    $$('.prose img').forEach(function (img) {
      img.addEventListener('click', function () { lb.firstElementChild.src = img.currentSrc || img.src; lb.hidden = false; });
    });
    lb.addEventListener('click', function () { lb.hidden = true; });
  }

  // ---- table of contents ----
  var tocLinks = $$('.toc a');
  if (tocLinks.length && 'IntersectionObserver' in window) {
    var byId = {};
    tocLinks.forEach(function (a) { byId[decodeURIComponent(a.hash.slice(1))] = a; });
    var visible = {};
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) { visible[e.target.id] = e.isIntersecting; });
      var first = tocLinks.map(function (a) { return decodeURIComponent(a.hash.slice(1)); }).filter(function (id) { return visible[id]; })[0];
      if (first) { tocLinks.forEach(function (a) { a.classList.remove('active'); }); byId[first].classList.add('active'); }
    }, { rootMargin: '-60px 0px -65% 0px' });
    Object.keys(byId).forEach(function (id) { var h = document.getElementById(id); if (h) io.observe(h); });
  }

  // ---- search ----
  var box = $('#search'), input = $('#search-input'), list = $('#search-results');
  var docs = null, results = [], sel = 0;
  function load() {
    if (docs) return Promise.resolve(docs);
    return fetch(root + 'search-index.json').then(function (r) { return r.json(); }).then(function (d) {
      docs = d.map(function (x) {
        x._t = x.t.toLowerCase(); x._h = (x.h || []).join(' ').toLowerCase(); x._x = x.x.toLowerCase(); return x;
      });
      return docs;
    });
  }
  function open() {
    if (!box) return;
    box.hidden = false; input.value = ''; render([]); input.focus(); load();
  }
  function close() { if (box) box.hidden = true; }
  function esc(s) { return s.replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
  function highlight(text, terms) {
    var out = esc(text);
    terms.forEach(function (t) {
      if (t.length < 2) return;
      out = out.replace(new RegExp('(' + t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'ig'), '<mark>$1</mark>');
    });
    return out;
  }
  function snippet(d, terms) {
    var i = -1;
    for (var k = 0; k < terms.length && i < 0; k++) i = d._x.indexOf(terms[k]);
    if (i < 0) return d.x.slice(0, 140) + (d.x.length > 140 ? '…' : '');
    var s = Math.max(0, i - 50);
    return (s > 0 ? '…' : '') + d.x.slice(s, s + 160) + (s + 160 < d.x.length ? '…' : '');
  }
  function search(q) {
    var terms = q.toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length || !docs) return [];
    var scored = [];
    docs.forEach(function (d) {
      var score = 0;
      for (var i = 0; i < terms.length; i++) {
        var t = terms[i], s = 0;
        if (d._t === t) s += 40;
        if (d._t.indexOf(t) === 0) s += 20; else if (d._t.indexOf(t) >= 0) s += 12;
        if (d._h.indexOf(t) >= 0) s += 6;
        var c = d._x.split(t).length - 1;
        if (c) s += Math.min(c, 10);
        if (!s) return; // every term must match somewhere
        score += s;
      }
      scored.push({ d: d, s: score });
    });
    scored.sort(function (a, b) { return b.s - a.s; });
    return scored.slice(0, 20).map(function (r) { return { d: r.d, terms: terms }; });
  }
  function render(rs) {
    results = rs; sel = 0;
    if (!input.value.trim()) { list.innerHTML = ''; return; }
    if (!rs.length) { list.innerHTML = '<li class="empty">No notes match “' + esc(input.value) + '”</li>'; return; }
    list.innerHTML = rs.map(function (r, i) {
      var d = r.d;
      return '<li' + (i === 0 ? ' class="sel"' : '') + ' role="option"><a href="' + root + d.u + '">' +
        '<span class="r-t">' + highlight(d.t, r.terms) + '</span>' +
        '<span class="r-s">' + esc(d.s + (d.g ? ' / ' + d.g : '')) + '</span>' +
        '<span class="r-x">' + highlight(snippet(d, r.terms), r.terms) + '</span></a></li>';
    }).join('');
  }
  function move(delta) {
    var items = $$('#search-results li[role="option"]');
    if (!items.length) return;
    items[sel].classList.remove('sel');
    sel = (sel + delta + items.length) % items.length;
    items[sel].classList.add('sel');
    items[sel].scrollIntoView({ block: 'nearest' });
  }
  if (box) {
    $$('[data-open-search]').forEach(function (b) { b.addEventListener('click', open); });
    $$('[data-close-search]').forEach(function (b) { b.addEventListener('click', close); });
    box.addEventListener('click', function (e) { if (e.target === box) close(); });
    input.addEventListener('input', function () { load().then(function () { render(search(input.value)); }); });
    input.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowDown') { e.preventDefault(); move(1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); move(-1); }
      else if (e.key === 'Enter') { var a = $('#search-results li.sel a'); if (a) location.href = a.href; }
      else if (e.key === 'Escape') { close(); }
    });
    document.addEventListener('keydown', function (e) {
      var typing = /INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName) || document.activeElement.isContentEditable;
      if ((e.key === '/' && !typing) || (e.key.toLowerCase() === 'k' && (e.metaKey || e.ctrlKey))) { e.preventDefault(); open(); }
      else if (e.key === 'Escape') { close(); document.body.classList.remove('nav-open'); if (lb) lb.hidden = true; }
    });
  }

  // ---- live reload (only under `notesync serve`) ----
  if (/^(localhost|127\.0\.0\.1|\[::1\])$/.test(location.hostname)) {
    var build = null;
    setInterval(function () {
      fetch('/__notesync/build', { cache: 'no-store' }).then(function (r) { return r.ok ? r.text() : null; }).then(function (b) {
        if (b === null) return;
        if (build !== null && b !== build) location.reload();
        build = b;
      }).catch(function () {});
    }, 1500);
  }
})();
