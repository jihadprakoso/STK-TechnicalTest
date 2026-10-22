"use client";

import Image from "next/image";
import { DragEvent, FormEvent, useCallback, useEffect, useMemo, useState } from "react";

type Menu = {
  id: string;
  name: string;
  parent_id: string | null;
  depth: number;
  position: number;
  children: Menu[];
};

type DialogState = { parent: Menu | null } | null;
type DropPosition = "before" | "inside" | "after";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    let message = `Request gagal (${response.status})`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // Preserve the HTTP status if the response isn't JSON.
    }
    throw new Error(message);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

function findMenu(nodes: Menu[], id: string): Menu | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    const child = findMenu(node.children, id);
    if (child) return child;
  }
  return null;
}

function allMenuIDs(nodes: Menu[]): string[] {
  return nodes.flatMap((node) => [node.id, ...allMenuIDs(node.children)]);
}

function siblingMenus(nodes: Menu[], parentID: string | null): Menu[] {
  if (parentID === null) return nodes;
  return findMenu(nodes, parentID)?.children ?? [];
}

type MenuTreeProps = {
  nodes: Menu[];
  expanded: Set<string>;
  selectedID: string;
  onToggle: (id: string) => void;
  onSelect: (menu: Menu) => void;
  onCreate: (parent: Menu) => void;
  onDelete: (menu: Menu) => void;
  onDrop: (sourceID: string, target: Menu, position: DropPosition) => void;
  onDragStateChange: (dragging: boolean) => void;
  dropTargetID: string;
  onDropTargetChange: (id: string) => void;
};

function MenuTree({
  nodes,
  expanded,
  selectedID,
  onToggle,
  onSelect,
  onCreate,
  onDelete,
  onDrop,
  onDragStateChange,
  dropTargetID,
  onDropTargetChange,
}: MenuTreeProps) {
  return (
    <ul className="tree-list">
      {nodes.map((menu) => {
        const hasChildren = menu.children.length > 0;
        const isOpen = expanded.has(menu.id);
        return (
          <li className="tree-item" key={menu.id}>
            <div
              className={`menu-row ${selectedID === menu.id ? "menu-row-selected" : ""} ${dropTargetID === menu.id ? "menu-row-drop-target" : ""}`}
              onDragOver={(event) => {
                if (event.dataTransfer.types.includes("text/plain")) {
                  event.preventDefault();
                  event.stopPropagation();
                  event.dataTransfer.dropEffect = "move";
                  onDropTargetChange(menu.id);
                }
              }}
              onDragLeave={(event) => {
                event.stopPropagation();
                if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
                  onDropTargetChange("");
                }
              }}
              onDrop={(event) => {
                event.preventDefault();
                event.stopPropagation();
                const sourceID = event.dataTransfer.getData("text/plain");
                if (sourceID && sourceID !== menu.id) {
                  const bounds = event.currentTarget.getBoundingClientRect();
                  const offset = (event.clientY - bounds.top) / bounds.height;
                  const position: DropPosition = offset < 0.25
                    ? "before"
                    : offset > 0.75
                      ? "after"
                      : "inside";
                  onDrop(sourceID, menu, position);
                }
                onDropTargetChange("");
              }}
            >
              <button
                aria-label={hasChildren ? `${isOpen ? "Ciutkan" : "Buka"} ${menu.name}` : "Menu tanpa anak"}
                className={`tree-toggle ${hasChildren ? "" : "tree-toggle-empty"}`}
                disabled={!hasChildren}
                onClick={() => onToggle(menu.id)}
                type="button"
              >
                {hasChildren && (
                  <svg
                    aria-hidden="true"
                    className={`tree-chevron ${isOpen ? "" : "tree-chevron-collapsed"}`}
                    viewBox="0 0 16 16"
                  >
                    <path d="m3 6 5 5 5-5" />
                  </svg>
                )}
              </button>
              <button
                aria-label={`Pindahkan ${menu.name}`}
                className="drag-handle"
                draggable
                onDragEnd={() => {
                  onDropTargetChange("");
                  onDragStateChange(false);
                }}
                onDragStart={(event) => {
                  event.dataTransfer.effectAllowed = "move";
                  event.dataTransfer.setData("text/plain", menu.id);
                  onDragStateChange(true);
                }}
                title="Seret untuk memindahkan"
                type="button"
              >
                ⠿
              </button>
              <button className="menu-select" onClick={() => onSelect(menu)} type="button">
                <span className="menu-folder" aria-hidden="true">{hasChildren ? "▰" : "▪"}</span>
                <span className="menu-name" title={menu.name}>{menu.name}</span>
              </button>
              <button
                aria-label={`Tambah submenu di bawah ${menu.name}`}
                className="add-child-button"
                onClick={() => onCreate(menu)}
                title="Tambah submenu"
                type="button"
              >
                <svg aria-hidden="true" className="tree-action-icon" viewBox="0 0 16 16">
                  <path d="M8 2.5v11M2.5 8h11" />
                </svg>
              </button>
              <button
                aria-label={`Hapus ${menu.name}`}
                className="delete-menu-button"
                onClick={() => onDelete(menu)}
                title="Hapus menu"
                type="button"
              >
                <svg aria-hidden="true" className="tree-action-icon tree-action-icon-delete" viewBox="0 0 16 16">
                  <path d="m3.5 3.5 9 9m0-9-9 9" />
                </svg>
              </button>
            </div>
            {hasChildren && isOpen && (
              <MenuTree
                nodes={menu.children}
                expanded={expanded}
                selectedID={selectedID}
                onToggle={onToggle}
                onSelect={onSelect}
                onCreate={onCreate}
                onDelete={onDelete}
                onDrop={onDrop}
                onDragStateChange={onDragStateChange}
                dropTargetID={dropTargetID}
                onDropTargetChange={onDropTargetChange}
              />
            )}
          </li>
        );
      })}
    </ul>
  );
}

export default function Home() {
  const [menus, setMenus] = useState<Menu[]>([]);
  const [selectedRootID, setSelectedRootID] = useState("");
  const [selectedID, setSelectedID] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [draggingMenu, setDraggingMenu] = useState(false);
  const [dropTargetID, setDropTargetID] = useState("");
  const [draftName, setDraftName] = useState<{ id: string; value: string } | null>(null);
  const [dialog, setDialog] = useState<DialogState>(null);
  const [newName, setNewName] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const acceptMenus = useCallback((result: Menu[]) => {
    setMenus(result);
    setExpanded((current) => {
      if (current.size) return current;
      return new Set(allMenuIDs(result));
    });
    setSelectedRootID((current) => result.some((root) => root.id === current) ? current : result[0]?.id ?? "");
    setSelectedID((current) => findMenu(result, current) ? current : result[0]?.id ?? "");
  }, []);

  const loadMenus = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      acceptMenus(await request<Menu[]>("/menus"));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat menu.");
    } finally {
      setLoading(false);
    }
  }, [acceptMenus]);

  useEffect(() => {
    let active = true;
    request<Menu[]>("/menus")
      .then((result) => {
        if (active) acceptMenus(result);
      })
      .catch((err: unknown) => {
        if (active) setError(err instanceof Error ? err.message : "Gagal memuat menu.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [acceptMenus]);

  const selectedMenu = useMemo(() => findMenu(menus, selectedID), [menus, selectedID]);
  const selectedRoot = menus.find((menu) => menu.id === selectedRootID) ?? null;
  const menuTree = selectedRoot ? [selectedRoot] : [];
  const selectedParent = selectedMenu?.parent_id
    ? findMenu(menus, selectedMenu.parent_id)
    : null;

  const name = selectedMenu
    ? draftName?.id === selectedMenu.id ? draftName.value : selectedMenu.name
    : "";

  function toggleMenu(id: string) {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function expandAll() {
    setExpanded(new Set(allMenuIDs(menuTree)));
  }

  function collapseAll() {
    setExpanded(new Set());
  }

  function openCreate(parent: Menu | null) {
    setError("");
    setNewName("");
    setDialog({ parent });
  }

  async function createMenu(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!dialog) return;
    setSaving(true);
    setError("");
    try {
      const created = await request<Menu>("/menus", {
        method: "POST",
        body: JSON.stringify({ name: newName, parent_id: dialog.parent?.id ?? null }),
      });
      const refreshed = await request<Menu[]>("/menus");
      acceptMenus(refreshed);
      setSelectedRootID(dialog.parent?.id ?? created.id);
      setSelectedID(created.id);
      if (dialog.parent) {
        setExpanded((current) => new Set(current).add(dialog.parent!.id));
      }
      setDialog(null);
      setNotice("Menu berhasil ditambahkan.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menambahkan menu.");
    } finally {
      setSaving(false);
    }
  }

  async function saveMenu(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedMenu) return;
    setSaving(true);
    setError("");
    setNotice("");
    try {
      await request<Menu>(`/menus/${selectedMenu.id}`, {
        method: "PUT",
        body: JSON.stringify({ name }),
      });
      setDraftName(null);
      await loadMenus();
      setNotice("Perubahan berhasil disimpan.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan perubahan.");
    } finally {
      setSaving(false);
    }
  }

  async function deleteMenu(menu: Menu) {
    const message = menu.children.length
      ? `Hapus "${menu.name}" dan seluruh submenu di bawahnya?`
      : `Hapus menu "${menu.name}"?`;
    if (!window.confirm(message)) return;
    setError("");
    setNotice("");
    try {
      await request<void>(`/menus/${menu.id}`, { method: "DELETE" });
      if (selectedID === menu.id || findMenu([menu], selectedID)) setSelectedID("");
      await loadMenus();
      setNotice("Menu berhasil dihapus.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menghapus menu.");
    }
  }

  async function moveMenu(sourceID: string, target: Menu, position: DropPosition) {
    const source = findMenu(menus, sourceID);
    if (!source || sourceID === target.id || findMenu([source], target.id)) return;
    setError("");
    setNotice("");
    try {
      if (position === "inside") {
        await request<Menu>(`/menus/${sourceID}/move`, {
          method: "PATCH",
          body: JSON.stringify({
            parent_id: target.id,
            position: target.children.length,
          }),
        });
        setExpanded((current) => new Set(current).add(target.id));
        setNotice(`Menu dipindahkan ke bawah ${target.name}.`);
      } else {
        const targetSiblings = siblingMenus(menus, target.parent_id);
        const orderedIDs = targetSiblings.map((menu) => menu.id).filter((id) => id !== sourceID);
        const targetIndex = orderedIDs.indexOf(target.id);
        if (targetIndex === -1) return;
        const positionIndex = targetIndex + (position === "after" ? 1 : 0);
        orderedIDs.splice(positionIndex, 0, sourceID);
        if (source.parent_id === target.parent_id) {
          await request<void>("/menus/reorder", {
            method: "PUT",
            body: JSON.stringify({ parent_id: target.parent_id, menu_ids: orderedIDs }),
          });
          setNotice("Urutan menu berhasil diperbarui.");
        } else {
          await request<Menu>(`/menus/${sourceID}/move`, {
            method: "PATCH",
            body: JSON.stringify({
              parent_id: target.parent_id,
              position: positionIndex,
            }),
          });
          setNotice("Menu dipindahkan.");
        }
      }
      await loadMenus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memindahkan menu.");
      await loadMenus();
    }
  }

  async function moveToTopLevel(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    const sourceID = event.dataTransfer.getData("text/plain");
    const source = findMenu(menus, sourceID);
    if (!source) return;
    setError("");
    setNotice("");
    try {
      if (source.parent_id === null) {
        const orderedIDs = menus.map((menu) => menu.id).filter((id) => id !== sourceID);
        orderedIDs.push(sourceID);
        await request<void>("/menus/reorder", {
          method: "PUT",
          body: JSON.stringify({ parent_id: null, menu_ids: orderedIDs }),
        });
        setNotice("Urutan menu utama berhasil diperbarui.");
      } else {
        await request<Menu>(`/menus/${sourceID}/move`, {
          method: "PATCH",
          body: JSON.stringify({ parent_id: null, position: menus.length }),
        });
        setSelectedRootID(sourceID);
        setSelectedID(sourceID);
        setNotice("Menu dipindahkan ke level utama.");
      }
      await loadMenus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memindahkan menu.");
      await loadMenus();
    }
  }

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-logo">
            <span aria-hidden="true" className="brand-symbol" />
            <span aria-label="Solusi Teknologi Kreatif" className="brand-name" role="img" />
          </div>
          <button aria-label="Buka navigasi" className="sidebar-menu-button" type="button">
            <Image alt="" height={24} src="/bar.svg" width={24} />
          </button>
        </div>
        <nav aria-label="Navigasi utama" className="side-navigation">
          <div className="nav-group">
            <button className="nav-item" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/systems.svg" alt="" width={24} height={24} />
</span>Systems</button>
            <button className="nav-item" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/system-code.svg" alt="" width={24} height={24} />
</span>System Code</button>
            <button className="nav-item" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/properties.svg" alt="" width={24} height={24} />
</span>Properties</button>
            <button className="nav-item nav-item-active" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/system-code-active.svg" alt="" width={24} height={24} />
</span>Menus</button>
            <button className="nav-item" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/properties.svg" alt="" width={24} height={24} />
</span>API List</button>
          </div>
          <button className="nav-item nav-standalone" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/folder2.svg" alt="" width={24} height={24} />
</span>Users &amp; Group</button>
          <button className="nav-item nav-standalone" type="button"><span className="nav-icon" aria-hidden="true">
  <Image src="/folder2.svg" alt="" width={24} height={24} />
</span>Competition</button>
        </nav>
      </aside>

      <section className="main-panel">
        <header className="topbar">
          <div className="breadcrumb">
            <svg aria-hidden="true" className="breadcrumb-folder" viewBox="0 0 24 24">
              <path d="M3 5.5A1.5 1.5 0 0 1 4.5 4h5l2 2H19.5A1.5 1.5 0 0 1 21 7.5v11a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18.5z" />
            </svg>
            <span>/</span>
            <strong>Menus</strong>
          </div>
          <button aria-label="Menu" className="mobile-menu-button" type="button">
            <Image src="/menu_open.svg" alt="" width={24} height={24} />
          </button>
        </header>

        <div className="content">
          <div className="page-heading">
            <span className="heading-icon" aria-hidden="true">
              <svg viewBox="0 0 36 36">
                <rect x="6" y="7" width="9" height="9" rx="1.5" />
                <circle cx="25" cy="11.5" r="4.5" />
                <rect x="6" y="21" width="9" height="9" rx="1.5" />
                <rect x="21" y="21" width="9" height="9" rx="1.5" />
              </svg>
            </span>
            <h1>Menus</h1>
          </div>

          {(error || notice) && (
            <div className={`feedback ${error ? "feedback-error" : "feedback-success"}`} role="status">
              <span>{error || notice}</span>
              <button aria-label="Tutup notifikasi" onClick={() => { setError(""); setNotice(""); }} type="button">×</button>
            </div>
          )}

          <div className="menu-picker">
            <label htmlFor="menu-root">Menu</label>
            <select
              id="menu-root"
              disabled={loading || menus.length === 0}
              onChange={(event) => {
                setSelectedRootID(event.target.value);
                setSelectedID(event.target.value);
                setExpanded((current) => new Set(current).add(event.target.value));
              }}
              value={selectedRootID}
            >
              {menus.length === 0 && <option value="">Belum ada menu</option>}
              {menus.map((menu) => <option key={menu.id} value={menu.id}>{menu.name}</option>)}
            </select>
          </div>

          <div className="manager-grid">
            <section aria-label="Pohon menu" className="tree-panel">
              <div className="tree-controls">
                <button className="expand-button" onClick={expandAll} type="button">Expand All</button>
                <button className="collapse-button" onClick={collapseAll} type="button">Collapse All</button>
                <button aria-label="Tambah menu utama" className="add-root-button" onClick={() => openCreate(null)} type="button">
                  <svg aria-hidden="true" className="tree-action-icon" viewBox="0 0 16 16">
                    <path d="M8 2.5v11M2.5 8h11" />
                  </svg>
                </button>
              </div>
              <div className="tree-content">
                {loading ? (
                  <div className="loading-state"><span className="spinner" /> Loading menus...</div>
                ) : menuTree.length ? (
                  <>
                    {draggingMenu && (
                      <div
                        className="root-drop-zone"
                        onDragOver={(event) => {
                          event.preventDefault();
                          event.dataTransfer.dropEffect = "move";
                        }}
                        onDrop={(event) => {
                          void moveToTopLevel(event);
                          setDropTargetID("");
                          setDraggingMenu(false);
                        }}
                      >
                        Drop here to move to Top level
                      </div>
                    )}
                    <MenuTree
                      nodes={menuTree}
                      expanded={expanded}
                      selectedID={selectedID}
                      onToggle={toggleMenu}
                      onSelect={(menu) => setSelectedID(menu.id)}
                      onCreate={openCreate}
                      onDelete={(menu) => void deleteMenu(menu)}
                      onDrop={(sourceID, target, position) => void moveMenu(sourceID, target, position)}
                      onDragStateChange={setDraggingMenu}
                      dropTargetID={dropTargetID}
                      onDropTargetChange={setDropTargetID}
                    />
                  </>
                ) : (
                  <div className="empty-state">
                    <span>No menus found</span>
                    <button onClick={() => openCreate(null)} type="button">Add your first menu</button>
                  </div>
                )}
              </div>
            </section>

            <section aria-label="Detail menu" className="detail-panel">
              <form onSubmit={saveMenu}>
                <label className="field-label" htmlFor="menu-id">Menu ID</label>
                <textarea
                  className="detail-input readonly-input"
                  id="menu-id"
                  readOnly
                  rows={2}
                  value={selectedMenu?.id ?? ""}
                />

                <label className="field-label" htmlFor="menu-depth">Depth</label>
                <input className="detail-input depth-input" id="menu-depth" readOnly value={selectedMenu?.depth ?? ""} />

                <label className="field-label" htmlFor="menu-parent">Parent Data</label>
                <input className="detail-input" id="menu-parent" readOnly value={selectedParent?.name ?? ""} />

                <label className="field-label" htmlFor="menu-name">Name</label>
                <input
                  className="detail-input"
                  disabled={!selectedMenu}
                  id="menu-name"
                  maxLength={100}
                  onChange={(event) => selectedMenu && setDraftName({ id: selectedMenu.id, value: event.target.value })}
                  value={name}
                />
                <button className="save-button" disabled={!selectedMenu || saving || !name.trim()} type="submit">
                  {saving ? "Saving..." : "Save"}
                </button>
              </form>
            </section>
          </div>
        </div>
      </section>

      {dialog && (
        <div className="modal-backdrop" onMouseDown={(event) => {
          if (event.target === event.currentTarget && !saving) setDialog(null);
        }}>
          <section aria-labelledby="dialog-title" aria-modal="true" className="menu-dialog" role="dialog">
            <button className="dialog-close" onClick={() => setDialog(null)} type="button" aria-label="Tutup">×</button>
            <h2 id="dialog-title">Add Menu</h2>
            <p>{dialog.parent ? `New menu under ${dialog.parent.name}` : "Add a top-level menu"}</p>
            <form onSubmit={createMenu}>
              <label className="field-label" htmlFor="new-menu-name">Name</label>
              <input
                autoFocus
                id="new-menu-name"
                maxLength={100}
                onChange={(event) => setNewName(event.target.value)}
                required
                value={newName}
              />
              {error && <div className="field-error">{error}</div>}
              <div className="dialog-actions">
                <button className="cancel-button" disabled={saving} onClick={() => setDialog(null)} type="button">Cancel</button>
                <button className="save-button" disabled={saving || !newName.trim()} type="submit">
                  {saving ? "Saving..." : "Save"}
                </button>
              </div>
            </form>
          </section>
        </div>
      )}
    </main>
  );
}
