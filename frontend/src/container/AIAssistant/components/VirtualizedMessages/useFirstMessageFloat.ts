import { RefObject, useLayoutEffect } from 'react';

const FIRST_MESSAGE_FLOAT_MS = 550;

// When each first message started floating, so a remounted list resumes the same flight.
const floatStarts = new Map<string, number>();

/** Floats the chat's first message up from just above the input to its place. */
export function useFirstMessageFloat(
	threadRef: RefObject<HTMLElement>,
	messageId: string | undefined,
	sentAt: number | null,
): void {
	useLayoutEffect(() => {
		const thread = threadRef.current;
		if (
			!messageId ||
			sentAt === null ||
			!thread ||
			Date.now() - (floatStarts.get(messageId) ?? sentAt) >=
				FIRST_MESSAGE_FLOAT_MS * 2 ||
			window.matchMedia('(prefers-reduced-motion: reduce)').matches
		) {
			return undefined;
		}

		let animation: Animation | undefined;
		const float = (): boolean => {
			const bubble = thread.querySelector<HTMLElement>(
				`[data-testid="ai-message-${messageId}"]`,
			);
			if (!bubble) {
				return false;
			}
			const rise =
				thread.getBoundingClientRect().bottom -
				bubble.getBoundingClientRect().bottom;
			animation = bubble.animate(
				[
					{ transform: `translateY(${Math.max(0, rise)}px)`, opacity: 0 },
					{ transform: 'translateY(0)', opacity: 1 },
				],
				{
					duration: FIRST_MESSAGE_FLOAT_MS,
					easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
					fill: 'backwards',
				},
			);
			if (!floatStarts.has(messageId)) {
				const now = Date.now();
				floatStarts.forEach((start, id) => {
					if (now - start >= FIRST_MESSAGE_FLOAT_MS * 2) {
						floatStarts.delete(id);
					}
				});
				floatStarts.set(messageId, now);
			}
			animation.currentTime = Date.now() - (floatStarts.get(messageId) ?? 0);
			return true;
		};

		if (float()) {
			return (): void => animation?.cancel();
		}
		// The list renders its rows a beat later; catch the bubble before it paints.
		const observer = new MutationObserver(() => {
			if (float()) {
				observer.disconnect();
			}
		});
		observer.observe(thread, { childList: true, subtree: true });
		const timeout = window.setTimeout(
			() => observer.disconnect(),
			FIRST_MESSAGE_FLOAT_MS,
		);
		return (): void => {
			observer.disconnect();
			window.clearTimeout(timeout);
			animation?.cancel();
		};
	}, [threadRef, messageId, sentAt]);
}
