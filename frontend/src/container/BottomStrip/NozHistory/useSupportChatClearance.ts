import { useLayoutEffect, useState } from 'react';

import {
	getSupportChatClearance,
	watchSupportChat,
} from '../SupportButton/supportChat';

// Outlasts the 0.2s ease, since the list only moves a frame or two after the support chat toggles.
const SLIDE_MS = 300;

interface SupportChatClearance {
	/** Room to leave at the viewport's right edge for the open support chat. */
	clearance: number;
	/** True while moving aside for the support chat, or back once it closes. */
	isSliding: boolean;
}

/** Keeps track of the room the support chat takes while `isActive`. */
export function useSupportChatClearance(
	isActive: boolean,
): SupportChatClearance {
	const [clearance, setClearance] = useState(0);
	const [isSliding, setIsSliding] = useState(false);

	// Before paint, so the list opens beside an open support chat instead of sliding there.
	useLayoutEffect(() => {
		if (!isActive) {
			return undefined;
		}
		let current = getSupportChatClearance();
		setClearance(current);

		let slideTimer: ReturnType<typeof setTimeout> | undefined;
		const stopWatching = watchSupportChat(() => {
			const next = getSupportChatClearance();
			if (next === current) {
				return;
			}
			current = next;
			setClearance(next);
			setIsSliding(true);
			clearTimeout(slideTimer);
			slideTimer = setTimeout(() => setIsSliding(false), SLIDE_MS);
		});

		return (): void => {
			stopWatching();
			clearTimeout(slideTimer);
			setIsSliding(false);
		};
	}, [isActive]);

	return { clearance, isSliding };
}
