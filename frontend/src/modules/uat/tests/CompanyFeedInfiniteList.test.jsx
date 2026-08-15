import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CompanyFeedInfiniteList } from "../components/CompanyFeedInfiniteList";

const { infiniteState, fetchNextPageMock } = vi.hoisted(() => ({
  infiniteState: { current: {} },
  fetchNextPageMock: vi.fn(),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useInfiniteQuery: () => ({ fetchNextPage: fetchNextPageMock, ...infiniteState.current }),
}));

const feedFixture = (id) => ({
  id,
  judul: `Feed ${id}`,
  konten_html: `<p>Konten ${id}</p>`,
  penulis: "HR Sintetis",
  published_at: "2026-08-01T02:00:00Z",
});

describe("CompanyFeedInfiniteList", () => {
  beforeEach(() => {
    fetchNextPageMock.mockReset();
  });

  it("shows loading, error, and empty states", () => {
    infiniteState.current = { isPending: true, isError: false };
    const { rerender } = render(<CompanyFeedInfiniteList />);
    expect(screen.getByText("Memuat company feed…")).toBeInTheDocument();

    infiniteState.current = { isPending: false, isError: true, error: { message: "Gagal." }, refetch: vi.fn() };
    rerender(<CompanyFeedInfiniteList />);
    expect(screen.getByRole("alert")).toHaveTextContent("Gagal");

    infiniteState.current = {
      isPending: false, isError: false, data: { pages: [{ items: [], meta: {} }] }, hasNextPage: false,
    };
    rerender(<CompanyFeedInfiniteList />);
    expect(screen.getByText("Belum ada informasi perusahaan.")).toBeInTheDocument();
  });

  it("flattens every loaded page into one list, 20 per page", () => {
    const pageOneItems = Array.from({ length: 20 }, (_, index) => feedFixture(`p1-${index}`));
    const pageTwoItems = [feedFixture("p2-0"), feedFixture("p2-1")];
    infiniteState.current = {
      isPending: false,
      isError: false,
      data: {
        pages: [
          { items: pageOneItems, meta: { page: 1, limit: 20, total_data: 22, total_page: 2 } },
          { items: pageTwoItems, meta: { page: 2, limit: 20, total_data: 22, total_page: 2 } },
        ],
      },
      hasNextPage: false,
      isFetchingNextPage: false,
    };
    render(<CompanyFeedInfiniteList />);

    expect(screen.getAllByRole("article")).toHaveLength(22);
    expect(screen.getByText("Semua feed sudah ditampilkan.")).toBeInTheDocument();
  });

  it("shows a loading indicator while fetching the next page and not the end-of-list footer", () => {
    infiniteState.current = {
      isPending: false,
      isError: false,
      data: { pages: [{ items: [feedFixture("p1-0")], meta: { page: 1, limit: 20, total_data: 21, total_page: 2 } }] },
      hasNextPage: true,
      isFetchingNextPage: true,
    };
    render(<CompanyFeedInfiniteList />);

    expect(screen.getByText("Memuat feed lainnya…")).toBeInTheDocument();
    expect(screen.queryByText("Semua feed sudah ditampilkan.")).not.toBeInTheDocument();
  });

  it("offers a retry action when loading the next page fails", () => {
    infiniteState.current = {
      isPending: false,
      isError: false,
      data: { pages: [{ items: [feedFixture("p1-0")], meta: { page: 1, limit: 20, total_data: 21, total_page: 2 } }] },
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: true,
    };
    render(<CompanyFeedInfiniteList />);

    expect(screen.getByText("Feed berikutnya belum dapat dimuat.")).toBeInTheDocument();
    screen.getByRole("button", { name: "Coba lagi" }).click();
    expect(fetchNextPageMock).toHaveBeenCalled();
  });
});
