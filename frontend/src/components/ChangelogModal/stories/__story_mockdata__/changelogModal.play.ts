/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { expect, screen, waitFor } from 'storybook/test';

import { isCloudShell } from '@/storybook/globals/appShellMocks';

const TITLE = /What’s New ⎯ Changelog : November 3, 2026/;

interface PlayContext {
	args: Record<string, unknown>;
}

export const showsCloudChangelog = async ({
	args,
}: PlayContext): Promise<void> => {
	await screen.findByRole('dialog', { name: TITLE });

	// The License and Banner controls decide the platform, which decides the footer.
	if (isCloudShell(args)) {
		await expect(
			screen.queryByRole('button', { name: /Skip for now/ }),
		).not.toBeInTheDocument();
		return;
	}

	await screen.findByRole('button', { name: /Skip for now/ });
};

export const showsSelfHostedChangelog = async (): Promise<void> => {
	await screen.findByRole('dialog', { name: TITLE });
	await waitFor(() =>
		expect(
			screen.getByRole('button', { name: /Update my workspace/ }),
		).toBeVisible(),
	);
};
