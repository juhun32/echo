export const MODEL_NAME = 'Xenova/all-MiniLM-L6-v2';

export type Message = {
	id: string;
	role: 'user' | 'assistant';
	text: string;
	source?: 'CACHE' | 'CLOUD';
	showCard?: boolean;
};

export type ChatSession = {
	id: string;
	title: string;
	messages: Message[];
	pinned?: boolean;
	folder?: string;
};

export type ChatApiResponse = {
	answer: string;
	source: 'CACHE' | 'CLOUD';
};

export type ExtractorOutput = {
	data: Float32Array;
};

export type FeatureExtractor = (
	text: string,
	options: { pooling: 'mean'; normalize: boolean }
) => Promise<ExtractorOutput>;
