import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// jsdom tidak mengimplementasikan document.execCommand/queryCommandState (rich-text editing
// perlu layout engine browser sungguhan). Tanpa stub ini, komponen WYSIWYG (react-simple-
// wysiwyg) melempar TypeError begitu tombol toolbar diklik di test. Stub hanya mencegah crash;
// stub tidak benar-benar memformat teks, jadi efek format tetap diverifikasi manual lewat
// Playwright terhadap Chromium nyata.
if (typeof document !== "undefined") {
  document.execCommand = document.execCommand ?? (() => false);
  document.queryCommandState = document.queryCommandState ?? (() => false);
}

// jsdom juga tidak mengimplementasikan IntersectionObserver, dipakai daftar company feed
// infinite scroll (Beranda) untuk mendeteksi sentinel di bawah daftar. Stub tidak pernah
// memicu callback; perilaku "load more saat scroll" sungguhan diverifikasi manual lewat
// Playwright terhadap Chromium nyata.
if (typeof window !== "undefined" && typeof window.IntersectionObserver === "undefined") {
  window.IntersectionObserver = class IntersectionObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

// jsdom juga tidak mengimplementasikan Element.scrollIntoView (perlu layout engine
// sungguhan), dipakai form Company Feed untuk membawa pengguna ke atas saat masuk mode edit.
if (typeof Element !== "undefined" && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}

afterEach(() => {
  cleanup();
});
