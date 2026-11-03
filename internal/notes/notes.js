// JavaScript for Automation (JXA) bridge to Apple Notes, run by notesync via
// `osascript -l JavaScript notes.js '<json request>'`.
//
// Privacy contract: only folders named in `include` (top level) are opened,
// and any folder - at any depth - whose trimmed, lower-cased name is in
// `exclude` is skipped without reading its notes. The "folders" mode reads
// folder names only, never note contents.
function run(argv) {
  const req = JSON.parse(argv[0]);
  const Notes = Application('Notes');
  const norm = (s) => String(s).trim().toLowerCase();
  const iso = (d) => (d && d.toISOString ? d.toISOString() : '');
  // parentFolder returns the containing folder, or null for a top-level folder.
  // (ObjectSpecifier.classOf does not work on Notes containers; pcls does.)
  const parentFolder = (f) => {
    try {
      const c = f.container();
      return c.properties().pcls === 'folder' ? c : null;
    } catch (e) { return null; }
  };

  if (req.mode === 'folders') {
    const out = [];
    Notes.accounts().forEach((acc) => {
      acc.folders().forEach((f) => {
        const p = parentFolder(f);
        const parent = p ? p.name() : '';
        out.push({ account: acc.name(), name: f.name(), parent: parent });
      });
    });
    return JSON.stringify(out);
  }

  if (req.mode === 'list') {
    const include = new Set((req.include || []).map(norm));
    const exclude = new Set((req.exclude || []).map(norm));
    const res = { notes: [], skipped: [], found: [] };
    const walk = (folder, path) => {
      const ids = folder.notes.id();
      const names = folder.notes.name();
      const created = folder.notes.creationDate();
      const modified = folder.notes.modificationDate();
      const locked = folder.notes.passwordProtected();
      for (let i = 0; i < ids.length; i++) {
        res.notes.push({
          id: ids[i], name: names[i], folder: path,
          created: iso(created[i]), modified: iso(modified[i]), locked: !!locked[i],
        });
      }
      folder.folders().forEach((sub) => {
        const n = sub.name();
        if (exclude.has(norm(n))) {
          res.skipped.push({ path: path.concat([n]).join('/'), reason: 'excluded folder' });
          return;
        }
        walk(sub, path.concat([n]));
      });
    };
    Notes.accounts().forEach((acc) => {
      if (req.account && acc.name() !== req.account) return;
      acc.folders().forEach((f) => {
        const n = f.name();
        if (!include.has(norm(n)) || exclude.has(norm(n))) return;
        if (parentFolder(f)) return; // nested folder that shares a name
        res.found.push(n);
        walk(f, [n]);
      });
    });
    return JSON.stringify(res);
  }

  if (req.mode === 'get') {
    const n = Notes.notes.byId(req.id);
    return JSON.stringify({ id: req.id, body: n.body() });
  }

  throw new Error('unknown mode: ' + req.mode);
}
