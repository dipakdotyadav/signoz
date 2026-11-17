import { useCallback, useMemo, useState } from 'react';
import { Button } from '@signozhq/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@signozhq/ui/popover';
import { Tooltip } from '@signozhq/ui/tooltip';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import {
	AIAssistantEvents,
	AIAssistantOpenSource,
} from 'container/AIAssistant/events';
import { normalizePage } from 'container/AIAssistant/hooks/useAIAssistantAnalyticsContext';
import {
	closeAIAssistant,
	openAIAssistant,
	useAIAssistantStore,
} from 'container/AIAssistant/store/useAIAssistantStore';
import { Conversation } from 'container/AIAssistant/types';
import { useIsAIAssistantEnabled } from 'hooks/useIsAIAssistantEnabled';
import { matchRoute } from 'lib/router/matchRoute';
import { useAppLocation } from 'lib/router/useAppLocation';
import { FolderClock } from '@signozhq/icons';

import { getSupportChatClearance } from '../SupportButton/supportChat';
import NozHistoryList from './NozHistoryList';

import styles from './NozHistory.module.scss';

const DAY_MS = 24 * 60 * 60 * 1000;

/** Recent Noz conversations from any page; AppLayout has already loaded them. */
function NozHistory(): JSX.Element | null {
	const { pathname } = useAppLocation();
	const isAIAssistantEnabled = useIsAIAssistantEnabled();
	const conversations = useAIAssistantStore((s) => s.conversations);
	const activeConversationId = useAIAssistantStore(
		(s) => s.activeConversationId,
	);
	const isDrawerOpen = useAIAssistantStore((s) => s.isDrawerOpen);
	const loadThread = useAIAssistantStore((s) => s.loadThread);
	const [isOpen, setIsOpen] = useState(false);
	const [supportChatClearance, setSupportChatClearance] = useState(0);

	// Chats saved on the server, newest first; unsent drafts don't count.
	const recent = useMemo(
		() =>
			Object.values(conversations)
				.filter((c) => c.threadId && !c.archived)
				.sort(
					(a, b) => (b.updatedAt ?? b.createdAt) - (a.updatedAt ?? a.createdAt),
				),
		[conversations],
	);

	const handleSelect = useCallback(
		(conversation: Conversation): void => {
			setIsOpen(false);
			if (conversation.threadId && conversation.id !== activeConversationId) {
				void logEvent(AIAssistantEvents.ThreadOpenedFromHistory, {
					threadId: conversation.threadId,
					threadAgeDays: Math.floor((Date.now() - conversation.createdAt) / DAY_MS),
					source: AIAssistantOpenSource.BottomStrip,
				});
				void loadThread(conversation.threadId);
			}
			if (!isDrawerOpen) {
				void logEvent(AIAssistantEvents.Opened, {
					source: AIAssistantOpenSource.BottomStrip,
					currentPage: normalizePage(pathname),
				});
				openAIAssistant();
			}
		},
		[activeConversationId, isDrawerOpen, loadThread, pathname],
	);

	// One Noz surface at a time: the list replaces the panel behind it.
	const handleOpenChange = useCallback((open: boolean): void => {
		setIsOpen(open);
		if (open) {
			setSupportChatClearance(getSupportChatClearance());
			closeAIAssistant();
		}
	}, []);

	// The full page lists its chats in its own sidebar.
	const isAIAssistantPage = !!matchRoute(pathname, ROUTES.AI_ASSISTANT, {
		exact: true,
	});

	if (!isAIAssistantEnabled || isAIAssistantPage || recent.length === 0) {
		return null;
	}

	return (
		<div className={styles.nozHistory} data-testid="bottom-strip-noz-history">
			<Popover open={isOpen} onOpenChange={handleOpenChange}>
				<Tooltip title="Recent conversations">
					<PopoverTrigger asChild>
						<Button
							variant="ghost"
							color="secondary"
							size="sm"
							icon
							aria-label="Recent conversations"
							testId="noz-history-trigger"
						>
							<FolderClock size={16} />
						</Button>
					</PopoverTrigger>
				</Tooltip>
				<PopoverContent
					className={styles.popover}
					side="top"
					align="end"
					sideOffset={4}
					// The support chat stays open over the strip, so the list opens left of it.
					collisionPadding={{ right: supportChatClearance }}
					sticky="always"
				>
					<NozHistoryList conversations={recent} onSelect={handleSelect} />
				</PopoverContent>
			</Popover>
		</div>
	);
}

export default NozHistory;
