import { act, renderHook } from 'tests/test-utils';

import { useSupportChatClearance } from '../useSupportChatClearance';

const CLOSED = 'PylonChat-chatWindowFrameContainer--closed';
// 1280 - 882 + 8px gap.
const CLEARANCE = 406;

function mountSupportChat(isOpen: boolean): HTMLElement {
	const element = document.createElement('div');
	element.className = 'PylonChat-chatWindowFrameContainer';
	element.classList.toggle(CLOSED, !isOpen);
	Object.defineProperties(element, {
		offsetLeft: { value: 882 },
		offsetTop: { value: 286 },
		offsetWidth: { value: 384 },
		offsetHeight: { value: 478 },
	});
	document.body.appendChild(element);
	return element;
}

describe('useSupportChatClearance', () => {
	const originalInnerWidth = window.innerWidth;

	beforeEach(() => {
		window.innerWidth = 1280;
	});

	afterEach(() => {
		window.innerWidth = originalInnerWidth;
		document
			.querySelectorAll('.PylonChat-chatWindowFrameContainer')
			.forEach((element) => element.remove());
	});

	it('opens clear of a support chat that is already open, without sliding', () => {
		mountSupportChat(true);

		const { result } = renderHook(() => useSupportChatClearance(true));

		expect(result.current).toStrictEqual({
			clearance: CLEARANCE,
			isSliding: false,
		});
	});

	it('slides aside as the support chat opens, and back as it closes', async () => {
		const supportChat = mountSupportChat(false);
		const { result } = renderHook(() => useSupportChatClearance(true));

		await act(async () => {
			supportChat.classList.remove(CLOSED);
		});

		expect(result.current).toStrictEqual({
			clearance: CLEARANCE,
			isSliding: true,
		});

		await act(async () => {
			supportChat.classList.add(CLOSED);
		});

		expect(result.current.clearance).toBe(0);
	});
});
