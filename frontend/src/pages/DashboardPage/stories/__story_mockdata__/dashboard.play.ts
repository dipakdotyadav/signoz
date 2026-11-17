/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { screen, userEvent, waitFor, within } from 'storybook/test';

import {
	canDeleteDashboard,
	canEditSections,
	clickSectionAction,
	type DashboardPageArgs,
	hasPanelSearch,
	openActionsMenu,
} from '../DashboardPage.stories.mocks';

interface PlayContext {
	canvasElement: HTMLElement;
	args: DashboardPageArgs;
}

export const expandPanelSearch = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!hasPanelSearch(args)) {
		return;
	}

	// Both the table and the log list render a search trigger; expand the first.
	const [trigger] = await within(canvasElement).findAllByTestId(
		'panel-header-search-trigger',
		{},
		{ timeout: 15000 },
	);
	await userEvent.click(trigger);
	const input = await screen.findByTestId('panel-header-search-input');

	await userEvent.type(input, 'error');
};

export const openDeleteDashboardDialog = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!canDeleteDashboard(args)) {
		return;
	}

	await openActionsMenu(canvasElement);
	// The row stays inert until its authz checks resolve, so one click can
	// land too early; keep clicking until the confirm dialog is up.
	await waitFor(
		async () => {
			const item = screen.queryByTestId('dashboard-action-delete');
			if (item) {
				await userEvent.click(item);
			}
			screen.getByText('This action cannot be undone.');
		},
		{ timeout: 10000 },
	);
};

export const openRenameSectionDialog = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!canEditSections(args)) {
		return;
	}

	await clickSectionAction(canvasElement, 'Rename section');
	await screen.findByRole('dialog', { name: /rename section/i });
};
