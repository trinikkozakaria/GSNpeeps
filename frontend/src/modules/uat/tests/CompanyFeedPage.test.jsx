import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CompanyFeedPage } from "../pages/CompanyFeedPage";

const { createMock, updateMock, deleteMock, mutationStates, feedsState, callOrder } = vi.hoisted(() => ({
  createMock: vi.fn(),
  updateMock: vi.fn(),
  deleteMock: vi.fn(),
  // [create, update, delete] tambahan state (isPending/isError/error) per mutation.
  mutationStates: { current: [{}, {}, {}] },
  feedsState: { current: {} },
  callOrder: { current: 0 },
}));

// CompanyFeedPage memanggil useMutation tiga kali per render dengan urutan tetap
// (create, update, delete) karena ketiganya top-level hook call tanpa kondisi; posisi
// panggilan ke-N modulo 3 selalu memetakan ke mutation yang sama di setiap render.
vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => feedsState.current,
  useMutation: () => {
    const mocks = [createMock, updateMock, deleteMock];
    const index = callOrder.current % 3;
    callOrder.current += 1;
    return { mutateAsync: mocks[index], ...mutationStates.current[index] };
  },
}));

const feedFixture = (overrides = {}) => ({
  id: "feed-1",
  judul: "Pengumuman Lama",
  konten_html: "<p>Konten lama</p>",
  penulis: "HR Sintetis",
  published_at: "2026-08-01T02:00:00Z",
  ...overrides,
});

const renderPage = (initialEntries = ["/app/company-feed"]) =>
  render(
    <MemoryRouter initialEntries={initialEntries}>
      <CompanyFeedPage />
    </MemoryRouter>,
  );

describe("CompanyFeedPage", () => {
  beforeEach(() => {
    createMock.mockReset().mockResolvedValue({});
    updateMock.mockReset().mockResolvedValue({});
    deleteMock.mockReset().mockResolvedValue({});
    mutationStates.current = [{}, {}, {}];
    callOrder.current = 0;
    feedsState.current = {
      data: { items: [feedFixture()], meta: { page: 1, limit: 20, total_data: 1, total_page: 1 } },
      isPending: false,
      isError: false,
    };
  });

  it("shows loading, error, and empty list states", () => {
    feedsState.current = { isPending: true, isError: false };
    const { rerender } = renderPage();
    expect(screen.getByText("Memuat company feed…")).toBeInTheDocument();

    feedsState.current = { isPending: false, isError: true, error: { message: "Gagal." }, refetch: vi.fn() };
    rerender(
      <MemoryRouter initialEntries={["/app/company-feed"]}>
        <CompanyFeedPage />
      </MemoryRouter>,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Gagal");

    feedsState.current = {
      data: { items: [], meta: { page: 1, limit: 20, total_data: 0, total_page: 0 } },
      isPending: false,
      isError: false,
    };
    rerender(
      <MemoryRouter initialEntries={["/app/company-feed"]}>
        <CompanyFeedPage />
      </MemoryRouter>,
    );
    expect(screen.getByText("Belum ada informasi perusahaan.")).toBeInTheDocument();
  });

  it("publishes trimmed WYSIWYG content and confirms success", async () => {
    const user = userEvent.setup();
    renderPage();
    const editor = screen.getByRole("textbox", { name: "Konten" });

    await user.type(screen.getByLabelText("Judul"), "  Informasi Kantor  ");
    editor.innerHTML = "<p>Agenda perusahaan</p>";
    editor.dispatchEvent(new Event("input", { bubbles: true }));
    await user.click(screen.getByRole("button", { name: "Terbitkan" }));

    await waitFor(() => expect(createMock).toHaveBeenCalledWith({
      judul: "Informasi Kantor",
      konten_html: "<p>Agenda perusahaan</p>",
    }));
    expect(await screen.findByText("Company feed berhasil diterbitkan.")).toBeInTheDocument();
  });

  it("renders pagination from the list meta and requests the next page", async () => {
    const user = userEvent.setup();
    feedsState.current = {
      data: { items: [feedFixture()], meta: { page: 1, limit: 20, total_data: 25, total_page: 2 } },
      isPending: false,
      isError: false,
    };
    renderPage();

    expect(screen.getByText("Halaman 1 dari 2 · 25 data")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Berikutnya" }));
    // Pagination menyimpan halaman di URL; komponen membaca ulang query lewat useSearchParams.
  });

  it("switches the form into edit mode, prefilled, and submits an update", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole("heading", { name: "Feed Baru" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Edit" }));

    expect(screen.getByRole("heading", { name: "Edit Feed" })).toBeInTheDocument();
    expect(screen.getByLabelText("Judul")).toHaveValue("Pengumuman Lama");
    expect(screen.getByRole("button", { name: "Simpan Perubahan" })).toBeInTheDocument();

    await user.clear(screen.getByLabelText("Judul"));
    await user.type(screen.getByLabelText("Judul"), "Pengumuman Baru");
    await user.click(screen.getByRole("button", { name: "Simpan Perubahan" }));

    await waitFor(() => expect(updateMock).toHaveBeenCalledWith({
      id: "feed-1",
      payload: { judul: "Pengumuman Baru", konten_html: "<p>Konten lama</p>" },
    }));
    expect(await screen.findByText("Company feed berhasil diperbarui.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Feed Baru" })).toBeInTheDocument();
  });

  it("cancels edit mode without submitting", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Edit" }));
    expect(screen.getByRole("heading", { name: "Edit Feed" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Batal edit" }));
    expect(screen.getByRole("heading", { name: "Feed Baru" })).toBeInTheDocument();
    expect(screen.getByLabelText("Judul")).toHaveValue("");
  });

  it("confirms before deleting and removes the feed", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    const dialog = screen.getByRole("alertdialog");
    expect(within(dialog).getByText(/Pengumuman Lama/)).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "Hapus feed" }));
    await waitFor(() => expect(deleteMock).toHaveBeenCalledWith("feed-1"));
  });

  it("cancelling the delete dialog does not call the mutation", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    await user.click(screen.getByRole("button", { name: "Batal" }));

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(deleteMock).not.toHaveBeenCalled();
  });

  // Toolbar dipakai apa adanya dari react-simple-wysiwyg (DefaultEditor, tidak di-wrap
  // children), bukan tombol kustom, setelah versi kustom sebelumnya terbukti tidak dapat
  // diandalkan. Tombol library tidak punya aria-label/teks (accessible name-nya jatuh ke
  // konten glyph seperti "𝐁", bukan atribut title), jadi dicari lewat getByTitle. jsdom
  // tidak mengimplementasikan document.execCommand (rich-text editing perlu layout engine
  // browser sungguhan), jadi efek format pada DOM diverifikasi manual lewat Playwright
  // terhadap Chromium nyata, bukan di sini.
  it("renders react-simple-wysiwyg's own unmodified toolbar buttons", async () => {
    const user = userEvent.setup();
    renderPage();

    for (const title of ["Bold", "Italic", "Underline", "Bullet list", "Numbered list"]) {
      const button = screen.getByTitle(title);
      expect(button.tagName).toBe("BUTTON");
      expect(button).toHaveClass("rsw-btn");
      // eslint-disable-next-line no-await-in-loop
      await user.click(button);
    }
  });
});
