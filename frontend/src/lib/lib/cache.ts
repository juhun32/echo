export type CacheEntry = {
	question: string;
	answer: string;
	source: string;
	createdAt: string;
};

export type CacheUse = {
	question: string;
	answer: string;
	source: string;
	timestamp: string;
	tokensSaved: number;
	energySavedWh: number;
	co2SavedG: number;
};

export type CacheStatsResponse = {
	uploading: boolean;
	lastUploadAt?: string;
	lastDownloadAt?: string;
	metrics: {
		cacheHits: number;
		localCacheHits: number;
		s3CacheHits: number;
		estimatedTokensSaved: number;
		energySavedWh: number;
		co2SavedG: number;
	};
	constants: {
		defaultKWhPer1KTokens: number;
		gridCO2gPerKWh: number;
		modelKWhPer1KTokens: Record<string, number>;
	};
	localRamCache: CacheEntry[];
	s3CacheUsed: CacheUse[];
};
