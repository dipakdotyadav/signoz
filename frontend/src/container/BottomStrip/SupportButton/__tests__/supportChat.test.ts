import { getSupportChatClearance } from '../supportChat';

describe('getSupportChatClearance', () => {
	const originalInnerWidth = window.innerWidth;

	afterEach(() => {
		window.innerWidth = originalInnerWidth;
		document.body.innerHTML = '';
	});

	it('keeps the open support chat, plus a gap, clear of the right edge', () => {
		window.innerWidth = 1280;
		const supportChat = document.createElement('div');
		supportChat.className = 'PylonChat-chatWindowFrameContainer';
		Object.defineProperty(supportChat, 'offsetLeft', { value: 882 });
		document.body.appendChild(supportChat);

		// 1280 - 882 + 8.
		expect(getSupportChatClearance()).toBe(406);
	});
});
