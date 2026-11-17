import {
	SUPPORT_CHAT_SELECTOR,
	supportChatObstacle,
} from './supportChatObstacle';

const SUPPORT_CHAT_TRIGGER_SELECTOR = '[data-support-chat-trigger]';

/** The space Noz leaves between itself and the support chat. */
const SUPPORT_CHAT_GAP = 8;

/** How much of the viewport's right side the open support chat takes, gap included; 0 when closed. */
export function getSupportChatClearance(): number {
	const supportChat = supportChatObstacle.getRect();
	return supportChat ? window.innerWidth - supportChat.x + SUPPORT_CHAT_GAP : 0;
}

/** Calls `onToggle` each time the support chat opens or closes; returns the unsubscribe. */
export function watchSupportChat(onToggle: () => void): () => void {
	const element = document.querySelector(SUPPORT_CHAT_SELECTOR);
	if (!element) {
		return (): void => undefined;
	}
	// Pylon keeps one callback per event, which supportChatObstacle holds, so watch the class instead.
	const observer = new MutationObserver(onToggle);
	observer.observe(element, { attributes: true, attributeFilter: ['class'] });
	return (): void => observer.disconnect();
}

/** Whether an event came from the support chat or the button that opens it. */
export function isSupportChatEvent(event: Event): boolean {
	return (
		event.target instanceof Element &&
		Boolean(
			event.target.closest(
				`${SUPPORT_CHAT_SELECTOR}, ${SUPPORT_CHAT_TRIGGER_SELECTOR}`,
			),
		)
	);
}
