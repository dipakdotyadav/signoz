/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { screen, userEvent, waitFor, within } from 'storybook/test';

import {
	type AIAssistantArgs,
	clickUntil,
	hasPendingClarification,
	showsConversationRows,
	untilLoaded,
} from '../AIAssistantPage.stories.mocks';
import { THREAD_TITLE } from './aiAssistant';

interface PlayContext {
	canvasElement: HTMLElement;
	args: AIAssistantArgs;
}

export const openClarificationSelect = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!hasPendingClarification(args)) {
		return;
	}
	const canvas = within(canvasElement);

	// Open from the keyboard, and retry: the message list remounts its items
	// while it measures the thread, which can close the popup again. The
	// "Other" checkbox in the Notify field carries the same text, so assert
	// on the popup's option role.
	await waitFor(async () => {
		const severityLabel = canvas.getByText('Severity');
		const field = severityLabel.closest('div');

		if (!field) {
			throw new Error('Severity field is not in the clarification form');
		}

		within(field).getByRole('combobox').focus();
		await userEvent.keyboard('{Enter}');
		await screen.findByRole(
			'option',
			{ name: 'Other (type your own)' },
			{ timeout: 1_000 },
		);
	}, untilLoaded);
};

export const renameConversation = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!showsConversationRows(args)) {
		return;
	}
	await clickUntil(async () => {
		const [actions] = await within(canvasElement).findAllByLabelText(
			/conversation actions/i,
			undefined,
			untilLoaded,
		);

		return actions;
	}, /^Rename$/);
	await userEvent.click(screen.getByText('Rename'));
	await within(canvasElement).findByDisplayValue(THREAD_TITLE);
};
