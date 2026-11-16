import { useEffect, useRef } from 'react';
import replySoundUrl from 'assets/sounds/noz-reply.wav';
import { useAIAssistantStore } from 'container/AIAssistant/store/useAIAssistantStore';

/** Chimes when a Noz reply lands in a conversation that isn't on screen. */
export function useNozReplySound(): void {
	const unreadReplyIds = useAIAssistantStore((s) => s.unreadReplyIds);
	const previousIdsRef = useRef(unreadReplyIds);
	const audioRef = useRef<HTMLAudioElement | null>(null);

	useEffect(() => {
		const previousIds = previousIdsRef.current;
		previousIdsRef.current = unreadReplyIds;
		if (unreadReplyIds.every((id) => previousIds.includes(id))) {
			return undefined;
		}

		// A chat on screen reads its reply in the same task, which re-runs this effect and cancels the chime.
		const timer = window.setTimeout(() => {
			audioRef.current ??= new Audio(replySoundUrl);
			audioRef.current.currentTime = 0;
			// Autoplay policy rejects play() until the user has interacted with the page.
			audioRef.current.play().catch(() => undefined);
		}, 0);

		return (): void => window.clearTimeout(timer);
	}, [unreadReplyIds]);
}
