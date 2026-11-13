import { useRef, useState } from 'react';
import { fireEvent, render, screen, userEvent } from 'tests/test-utils';

import ComposerCaret from '../ComposerCaret';

function Composer(): JSX.Element {
	const textareaRef = useRef<HTMLTextAreaElement>(null);
	const [value, setValue] = useState('');
	return (
		<>
			<textarea
				ref={textareaRef}
				data-testid="composer"
				value={value}
				onChange={(event): void => setValue(event.target.value)}
			/>
			<button type="button" data-testid="elsewhere">
				Elsewhere
			</button>
			<ComposerCaret textareaRef={textareaRef} value={value} />
		</>
	);
}

describe('ComposerCaret', () => {
	beforeEach(() => {
		// jsdom has no layout; the caret only shows inside the textarea's visible height.
		jest
			.spyOn(HTMLElement.prototype, 'offsetHeight', 'get')
			.mockImplementation(function offsetHeight(this: HTMLElement) {
				return this instanceof HTMLTextAreaElement ? 40 : 0;
			});
	});

	afterEach(() => {
		jest.restoreAllMocks();
	});

	it('draws its caret while the composer has focus', async () => {
		const user = userEvent.setup({ pointerEventsCheck: 0 });
		render(<Composer />);

		await user.click(screen.getByTestId('composer'));

		expect(screen.getByTestId('ai-composer-caret')).toBeInTheDocument();
	});

	it('hides its caret once the composer loses focus', async () => {
		const user = userEvent.setup({ pointerEventsCheck: 0 });
		render(<Composer />);
		await user.click(screen.getByTestId('composer'));

		await user.click(screen.getByTestId('elsewhere'));

		expect(screen.queryByTestId('ai-composer-caret')).not.toBeInTheDocument();
	});

	it('hides its caret while text is selected', async () => {
		const user = userEvent.setup({ pointerEventsCheck: 0 });
		render(<Composer />);
		const composer = screen.getByTestId<HTMLTextAreaElement>('composer');
		await user.type(composer, 'checkout latency');

		composer.setSelectionRange(0, 8);
		// jsdom fires no selectionchange for setSelectionRange.
		fireEvent(document, new Event('selectionchange'));

		expect(screen.queryByTestId('ai-composer-caret')).not.toBeInTheDocument();
	});
});
