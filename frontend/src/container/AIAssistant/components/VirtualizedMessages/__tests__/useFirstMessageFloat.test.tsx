import { renderHook } from 'tests/test-utils';

import { useFirstMessageFloat } from '../useFirstMessageFloat';

interface FakeAnimation {
	currentTime: number | null;
	cancel: jest.Mock;
}

const NOW = 50_000;
const animations: FakeAnimation[] = [];
const animate = jest.fn((): FakeAnimation => {
	const animation: FakeAnimation = { currentTime: null, cancel: jest.fn() };
	animations.push(animation);
	return animation;
});

function threadWithMessage(messageId: string): { current: HTMLDivElement } {
	const thread = document.createElement('div');
	const bubble = document.createElement('div');
	bubble.dataset.testid = `ai-message-${messageId}`;
	thread.appendChild(bubble);
	document.body.appendChild(thread);
	return { current: thread };
}

function reduceMotion(reduce: boolean): void {
	jest
		.spyOn(window, 'matchMedia')
		.mockImplementation((query: string): MediaQueryList => ({
			matches: reduce && query === '(prefers-reduced-motion: reduce)',
			media: query,
			onchange: null,
			addListener: jest.fn(),
			removeListener: jest.fn(),
			addEventListener: jest.fn(),
			removeEventListener: jest.fn(),
			dispatchEvent: jest.fn(),
		}));
}

describe('useFirstMessageFloat', () => {
	beforeAll(() => {
		Object.defineProperty(Element.prototype, 'animate', {
			configurable: true,
			value: animate,
		});
	});

	beforeEach(() => {
		jest.spyOn(Date, 'now').mockReturnValue(NOW);
		reduceMotion(false);
		animations.length = 0;
		animate.mockClear();
	});

	afterEach(() => {
		jest.restoreAllMocks();
		document.body.innerHTML = '';
	});

	it('floats the first message up into place', () => {
		const threadRef = threadWithMessage('first');

		renderHook(() => useFirstMessageFloat(threadRef, 'first', NOW));

		expect(animate).toHaveBeenCalledTimes(1);
		expect(animate).toHaveBeenCalledWith(
			[
				{ transform: 'translateY(0px)', opacity: 0 },
				{ transform: 'translateY(0)', opacity: 1 },
			],
			expect.objectContaining({ duration: 550 }),
		);
	});

	it('resumes the same flight after the list remounts', () => {
		const threadRef = threadWithMessage('resumed');
		const { unmount } = renderHook(() =>
			useFirstMessageFloat(threadRef, 'resumed', NOW),
		);
		unmount();
		jest.spyOn(Date, 'now').mockReturnValue(NOW + 200);

		renderHook(() => useFirstMessageFloat(threadRef, 'resumed', NOW));

		expect(animations[1].currentTime).toBe(200);
	});

	it('stays put with reduced motion', () => {
		reduceMotion(true);
		const threadRef = threadWithMessage('still');

		renderHook(() => useFirstMessageFloat(threadRef, 'still', NOW));

		expect(animate).not.toHaveBeenCalled();
	});
});
