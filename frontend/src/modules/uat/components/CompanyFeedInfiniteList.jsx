import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { Button } from "../../../components/ui/Button";
import { feedsRequest } from "../api/uat-api";
import { FeedCard } from "./FeedCard";

export const FEED_PAGE_SIZE = 20;

/**
 * Beranda menampilkan company feed dengan infinite scroll: 20 feed per load, halaman
 * berikutnya dimuat otomatis saat sentinel di bawah daftar terlihat di viewport.
 */
export const CompanyFeedInfiniteList = () => {
  const feeds = useInfiniteQuery({
    queryKey: ["company-feed", "infinite"],
    queryFn: ({ pageParam, signal }) => feedsRequest({ page: pageParam, limit: FEED_PAGE_SIZE }, signal),
    initialPageParam: 1,
    getNextPageParam: (lastPage) =>
      lastPage.meta.page < lastPage.meta.total_page ? lastPage.meta.page + 1 : undefined,
  });
  const sentinelRef = useRef(null);

  useEffect(() => {
    const node = sentinelRef.current;
    if (!node || !feeds.hasNextPage || typeof IntersectionObserver === "undefined") return undefined;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting && !feeds.isFetchingNextPage) {
        feeds.fetchNextPage();
      }
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, [feeds.hasNextPage, feeds.isFetchingNextPage, feeds.fetchNextPage]);

  if (feeds.isPending) return <p role="status">Memuat company feed…</p>;
  if (feeds.isError) {
    return (
      <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
        <p>Company feed belum dapat dimuat. {feeds.error?.message}</p>
        <Button className="mt-3" variant="secondary" onClick={() => feeds.refetch()}>Coba lagi</Button>
      </div>
    );
  }

  const items = feeds.data.pages.flatMap((page) => page.items);
  if (items.length === 0) {
    return <p className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5 text-slate-600">Belum ada informasi perusahaan.</p>;
  }

  return (
    <div className="grid gap-4">
      {items.map((feed) => (
        <FeedCard key={feed.id} feed={feed} />
      ))}
      <div ref={sentinelRef} aria-hidden="true" className="h-1" />
      <div aria-live="polite">
        {feeds.isFetchingNextPage && (
          <p role="status" className="text-center text-sm text-slate-500">Memuat feed lainnya…</p>
        )}
        {feeds.isFetchNextPageError && (
          <div role="alert" className="text-center text-sm text-red-700">
            <p>Feed berikutnya belum dapat dimuat.</p>
            <Button className="mt-2" variant="secondary" onClick={() => feeds.fetchNextPage()}>
              Coba lagi
            </Button>
          </div>
        )}
        {!feeds.hasNextPage && items.length > FEED_PAGE_SIZE && (
          <p className="text-center text-sm text-slate-400">Semua feed sudah ditampilkan.</p>
        )}
      </div>
    </div>
  );
};
