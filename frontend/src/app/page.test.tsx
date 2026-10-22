import React from "react";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Menu } from "./types";
import Home from "./page";

vi.mock("next/image", () => ({
  default: ({ alt }: React.ImgHTMLAttributes<HTMLImageElement>) => (
    <span aria-label={alt || undefined} aria-hidden={!alt} role={alt ? "img" : undefined} />
  ),
}));

const rootID = "9465f973-2aa8-473d-b391-9b3027f2a2bb";
const childID = "2917550a-6342-405a-b4eb-cf3870d6ae7d";

function createMenus(): Menu[] {
  return [{
    id: rootID,
    name: "Root",
    parent_id: null,
    depth: 0,
    position: 0,
    children: [{
      id: childID,
      name: "Child",
      parent_id: rootID,
      depth: 1,
      position: 0,
      children: [],
    }],
  }];
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

describe("menu manager", () => {
  let menus: Menu[];
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    menus = createMenus();
    fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";

      if (url.endsWith("/menus") && method === "GET") return jsonResponse(menus);
      if (url.endsWith("/menus") && method === "POST") {
        const body = JSON.parse(String(init?.body)) as { name: string; parent_id: string | null };
        const created: Menu = {
          id: "81057828-f358-42fd-8bb8-95ebc98cd79d",
          name: body.name,
          parent_id: body.parent_id,
          depth: body.parent_id ? 1 : 0,
          position: body.parent_id ? menus[0].children.length : menus.length,
          children: [],
        };
        if (body.parent_id) menus[0].children.push(created);
        else menus.push(created);
        return jsonResponse(created, 201);
      }
      if (url.endsWith(`/menus/${rootID}`) && method === "PUT") {
        const body = JSON.parse(String(init?.body)) as { name: string };
        menus[0].name = body.name;
        return jsonResponse(menus[0]);
      }
      if (url.endsWith(`/menus/${childID}`) && method === "DELETE") {
        menus[0].children = [];
        return jsonResponse(null, 204);
      }

      return jsonResponse({ error: "not found" }, 404);
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("loads the hierarchy and expands or collapses its children", async () => {
    const user = userEvent.setup();
    render(<Home />);

    expect(await screen.findByRole("button", { name: "Ciutkan Root" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Child" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Collapse All" }));
    expect(screen.queryByRole("button", { name: "Child" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Expand All" }));
    expect(screen.getByRole("button", { name: "Child" })).toBeInTheDocument();
  });

  it("creates a child menu and refreshes the tree", async () => {
    const user = userEvent.setup();
    render(<Home />);

    await screen.findByRole("button", { name: "Ciutkan Root" });
    await user.click(screen.getByRole("button", { name: "Tambah submenu di bawah Root" }));

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Name"), "New child");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    expect(await screen.findByText("New child")).toBeInTheDocument();
    expect(await screen.findByRole("status")).toHaveTextContent("Menu berhasil ditambahkan.");
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/menus$/),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("searches menus while keeping matching ancestors visible", async () => {
    const user = userEvent.setup();
    render(<Home />);

    await screen.findByRole("button", { name: "Ciutkan Root" });
    await user.type(screen.getByRole("searchbox", { name: "Search menus" }), "child");

    expect(screen.getByRole("button", { name: "Root" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Child" })).toBeInTheDocument();

    await user.clear(screen.getByRole("searchbox", { name: "Search menus" }));
    expect(screen.getByRole("button", { name: "Child" })).toBeInTheDocument();
  });

  it("creates a root menu", async () => {
    const user = userEvent.setup();
    render(<Home />);

    await screen.findByRole("button", { name: "Ciutkan Root" });
    await user.click(screen.getByRole("button", { name: "Tambah menu utama" }));
    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Name"), "New root");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    expect(await screen.findByRole("button", { name: "New root" })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/menus$/),
      expect.objectContaining({ method: "POST", body: JSON.stringify({ name: "New root", parent_id: null }) }),
    );
  });

  it("saves an edited menu name", async () => {
    const user = userEvent.setup();
    render(<Home />);

    await screen.findByRole("button", { name: "Ciutkan Root" });
    const nameInput = screen.getByLabelText("Name");
    await user.clear(nameInput);
    await user.type(nameInput, "Updated root");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(menus[0].name).toBe("Updated root"));
    expect(await screen.findByRole("status")).toHaveTextContent("Perubahan berhasil disimpan.");
  });

  it("deletes a menu after confirmation", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("confirm", vi.fn(() => true));
    render(<Home />);

    await screen.findByRole("button", { name: "Ciutkan Root" });
    await user.click(screen.getByRole("button", { name: "Hapus Child" }));

    await waitFor(() => expect(menus[0].children).toHaveLength(0));
    expect(screen.queryByRole("button", { name: "Child" })).not.toBeInTheDocument();
    expect(window.confirm).toHaveBeenCalledOnce();
  });
});
