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

afterEach(() => {
  cleanup();
});
