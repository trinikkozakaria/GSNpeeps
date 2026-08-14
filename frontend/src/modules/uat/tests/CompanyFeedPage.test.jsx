import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CompanyFeedList, CompanyFeedPage } from "../pages/CompanyFeedPage";

const { createMock, createState, feedsState } = vi.hoisted(() => ({
  createMock: vi.fn(),
  createState: { current: {} },
  feedsState: { current: {} },
}));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => feedsState.current,
  useMutation: () => ({ mutateAsync: createMock, ...createState.current }),
}));

describe("CompanyFeedPage", () => {
  beforeEach(() => {
    createMock.mockReset();
    createMock.mockResolvedValue({});
    createState.current = { isPending: false, isError: false };
    feedsState.current = { data: [], isPending: false, isError: false };
  });

  it("shows loading, error, and empty feed states", () => {
    feedsState.current = { isPending: true, isError: false };
    const { rerender } = render(<CompanyFeedList />);
    expect(screen.getByRole("status")).toHaveTextContent("Memuat company feed");

    feedsState.current = { isPending: false, isError: true, error: { message: "Gagal." }, refetch: vi.fn() };
    rerender(<CompanyFeedList />);
    expect(screen.getByRole("alert")).toHaveTextContent("Gagal");

    feedsState.current = { data: [], isPending: false, isError: false };
    rerender(<CompanyFeedList />);
    expect(screen.getByText("Belum ada informasi perusahaan.")).toBeInTheDocument();
  });

  it("publishes trimmed WYSIWYG content and confirms success", async () => {
    const user = userEvent.setup();
    render(<CompanyFeedPage />);
    const editor = screen.getByRole("textbox", { name: "Konten" });

    await user.type(screen.getByLabelText("Judul"), "  Informasi Kantor  ");
    editor.innerHTML = "<p>Agenda perusahaan</p>";
    fireEvent.input(editor);
    await user.click(screen.getByRole("button", { name: "Terbitkan" }));

    await waitFor(() => expect(createMock).toHaveBeenCalledWith({
      judul: "Informasi Kantor",
      konten_html: "<p>Agenda perusahaan</p>",
    }));
    expect(await screen.findByText("Company feed berhasil diterbitkan.")).toBeInTheDocument();
  });
});
