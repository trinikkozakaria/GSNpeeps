# Accessibility and responsive behavior

Target WCAG 2.2 AA behavior for critical GSNpeeps workflows.

## Contents

[Semantics](#semantics) · [Keyboard and focus](#keyboard-and-focus) · [Forms](#forms) ·
[Async feedback](#async-feedback) · [Color and motion](#color-and-motion) ·
[Responsive behavior](#responsive-behavior) ·
[GSNpeeps-specific interactions](#gsnpeeps-specific-interactions) ·
[Verification](#verification)

## Semantics

- Use landmarks: header, nav, main, aside, footer where appropriate.
- Provide a skip link.
- Keep one meaningful page `h1` and logical heading order.
- Use Button for actions and Link for navigation.
- Use native table/form/dialog semantics before custom roles.
- Give icon-only controls an accessible name.
- Mark decorative icons/images appropriately.

## Keyboard and focus

- Make every action reachable and operable without a mouse.
- Keep focus indicators visible and high contrast.
- Trap focus inside modal dialogs and restore it to the trigger.
- Move focus to page heading after route changes where necessary.
- Focus first invalid field/error summary after failed submission.
- Do not implement keyboard traps in tables, menus, camera, or uploads.
- Support Escape only where it does not discard important input unexpectedly.

## Forms

- Use visible labels; placeholder is not a label.
- Associate descriptions and errors.
- Expose required/invalid state programmatically.
- Use correct autocomplete for login and employee fields where safe.
- Keep validation messages in Bahasa Indonesia and corrective.
- Do not announce every keystroke as an error.

## Async feedback

- Use a polite live region for meaningful completion/error updates.
- Use `aria-busy` or equivalent on changing regions.
- Keep button names stable; add a separate busy indicator/text.
- Avoid noisy announcements during polling.
- Announce pagination/filter results when useful.

## Color and motion

- Meet AA contrast for text, controls, focus, and status.
- Pair color with text, icon, pattern, or label.
- Respect reduced-motion preference.
- Avoid flashing and non-essential animation.
- Keep charts distinguishable without color alone.

## Responsive behavior

- Start mobile-first.
- Prevent page-level horizontal overflow.
- Support 200% zoom without loss of content/action.
- Use readable touch targets and spacing.
- Let long Indonesian labels and employee names wrap.
- Choose an intentional mobile table strategy.
- Keep dialogs/forms usable with the on-screen keyboard.

## GSNpeeps-specific interactions

### Camera

- Explain camera purpose before requesting permission.
- Name capture, retake, switch, and fallback controls.
- Provide non-live fallback upload.
- Do not rely on preview image alone; describe capture state.
- Stop the stream when no longer needed.

### Geolocation

- Explain why location is needed.
- Present locating, denied, unavailable, timeout, in-radius, and out-of-radius states.
- Do not express radius result by color alone.
- Do not imply browser-calculated time/radius is authoritative.

### Charts and metrics

- Provide descriptive titles and summaries.
- Include a textual/table equivalent for material values.
- Avoid inaccessible hover-only tooltips.
- Distinguish no data from zero.

### Approval timeline

- Use an ordered list with actor/stage/status/time text.
- Mark current stage in text.
- Keep decision buttons after enough context for informed action.

### Notifications

- Use meaningful link text and unread text/state.
- Avoid announcing every polling refresh.
- Maintain focus after read/dismiss.

## Verification

For critical routes:

1. Navigate keyboard-only.
2. Verify screen-reader names/relationships.
3. Test 200% zoom and narrow viewport.
4. Test reduced motion and high contrast where supported.
5. Run the approved automated accessibility checker.
6. Manually inspect issues automation cannot detect.

Automated checks do not replace keyboard and screen-reader-oriented review.
