/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { MessagingQueuesViewTypeOptions } from 'pages/MessagingQueues/MessagingQueuesUtils';
import { screen, userEvent, within } from 'storybook/test';

import type { PageStoryArgs } from '@/storybook/runtime/resolveStory';

import type { kafkaDetailMocks } from '../MQDetailPage.stories.mocks';

interface PlayContext {
	canvasElement: HTMLElement;
	args: PageStoryArgs<typeof kafkaDetailMocks>;
}

const untilLoaded = { timeout: 15_000 };

/** Only the drop rate view carries the evaluation interval selector. */
export const openCustomInterval = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (args.view !== MessagingQueuesViewTypeOptions.DropRate) {
		return;
	}

	await userEvent.click(
		await within(canvasElement).findByText('10ms', undefined, untilLoaded),
	);
	await screen.findByPlaceholderText(
		'Enter custom time (ms)',
		undefined,
		untilLoaded,
	);
};
