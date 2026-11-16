import getLocalStorageKey from 'api/browser/localstorage/get';
import setLocalStorageKey from 'api/browser/localstorage/set';
import { useAIAssistantStore } from 'container/AIAssistant/store/useAIAssistantStore';
import type { ConversationStreamState } from 'container/AIAssistant/types';

import {
	NOZ_OPEN_CONVERSATIONS_STORAGE_KEY,
	useNozOpenConversationsStore,
} from '../useNozOpenConversationsStore';

const initialAIState = useAIAssistantStore.getState();
const initialTabsState = useNozOpenConversationsStore.getState();

const streaming: ConversationStreamState = {
	isStreaming: true,
	streamingContent: '',
	streamingStatus: 'thinking',
	streamingEvents: [],
	streamingMessageId: null,
	streamingActions: null,
	pendingApproval: null,
	pendingClarification: null,
	startedAt: 0,
};

function seedChats(
	ids: string[],
	busy: { unread?: string[]; streaming?: string[] } = {},
): void {
	useAIAssistantStore.setState((s) => {
		ids.forEach((id) => {
			s.conversations[id] = { id, threadId: id, messages: [], createdAt: 0 };
		});
		s.unreadReplyIds = busy.unread ?? [];
		(busy.streaming ?? []).forEach((id) => {
			s.streams[id] = streaming;
		});
	});
}

function openAll(ids: string[]): void {
	ids.forEach((id) => useNozOpenConversationsStore.getState().open(id));
}

const tabs = (): string[] => useNozOpenConversationsStore.getState().threadIds;

describe('useNozOpenConversationsStore', () => {
	afterEach(() => {
		useAIAssistantStore.setState(initialAIState, true);
		useNozOpenConversationsStore.setState(initialTabsState, true);
		localStorage.clear();
	});

	it('opens a tab once, in open order', () => {
		openAll(['a', 'b', 'a']);

		expect(tabs()).toStrictEqual(['a', 'b']);
	});

	it('past the cap drops the oldest tab with no unseen reply or running stream', () => {
		seedChats(['a', 'b', 'c', 'd', 'e', 'f'], {
			unread: ['a'],
			streaming: ['b'],
		});

		openAll(['a', 'b', 'c', 'd', 'e', 'f']);

		expect(tabs()).toStrictEqual(['a', 'b', 'd', 'e', 'f']);
	});

	it('drops the oldest tab when every tab is busy', () => {
		seedChats(['a', 'b', 'c', 'd', 'e', 'f'], {
			unread: ['a', 'b', 'c', 'd', 'e'],
		});

		openAll(['a', 'b', 'c', 'd', 'e', 'f']);

		expect(tabs()).toStrictEqual(['b', 'c', 'd', 'e', 'f']);
	});

	it('closes and prunes tabs', () => {
		openAll(['a', 'b', 'c', 'd']);

		useNozOpenConversationsStore.getState().close('b');
		useNozOpenConversationsStore.getState().prune(['c', 'missing']);

		expect(tabs()).toStrictEqual(['a', 'd']);
	});

	it('saves the tabs as soon as they change', () => {
		openAll(['a', 'b']);

		expect(
			JSON.parse(getLocalStorageKey(NOZ_OPEN_CONVERSATIONS_STORAGE_KEY) ?? '{}'),
		).toStrictEqual({ state: { threadIds: ['a', 'b'] }, version: 1 });
	});

	it('restores only string ids from storage, newest five', async () => {
		setLocalStorageKey(
			NOZ_OPEN_CONVERSATIONS_STORAGE_KEY,
			JSON.stringify({
				state: { threadIds: ['1', '2', 3, '4', '5', '6', null, '7'] },
				version: 1,
			}),
		);

		await useNozOpenConversationsStore.persist.rehydrate();

		expect(tabs()).toStrictEqual(['2', '4', '5', '6', '7']);
	});

	it('restores no tabs from a stored value that is not a list', async () => {
		useNozOpenConversationsStore.setState({ threadIds: ['open'] });
		setLocalStorageKey(
			NOZ_OPEN_CONVERSATIONS_STORAGE_KEY,
			JSON.stringify({ state: { threadIds: 'a' }, version: 1 }),
		);

		await useNozOpenConversationsStore.persist.rehydrate();

		expect(tabs()).toStrictEqual([]);
	});
});
