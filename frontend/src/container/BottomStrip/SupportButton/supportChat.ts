import { supportChatObstacle } from './supportChatObstacle';

const SUPPORT_CHAT_GAP = 8;

/** How much of the viewport's right side the open support chat takes, gap included; 0 when closed. */
export function getSupportChatClearance(): number {
	const supportChat = supportChatObstacle.getRect();
	return supportChat ? window.innerWidth - supportChat.x + SUPPORT_CHAT_GAP : 0;
}
