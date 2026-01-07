export namespace backend {
	
	export class AnalysisResult {
	    filePath: string;
	    fileName: string;
	    isTrueLossless: boolean;
	    confidence: number;
	    spectrumCutoff: number;
	    expectedCutoff: number;
	    verdict: string;
	    verdictLabel: string;
	    details: string;
	    sampleRate: number;
	    bitDepth: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.isTrueLossless = source["isTrueLossless"];
	        this.confidence = source["confidence"];
	        this.spectrumCutoff = source["spectrumCutoff"];
	        this.expectedCutoff = source["expectedCutoff"];
	        this.verdict = source["verdict"];
	        this.verdictLabel = source["verdictLabel"];
	        this.details = source["details"];
	        this.sampleRate = source["sampleRate"];
	        this.bitDepth = source["bitDepth"];
	    }
	}
	export class Config {
	    tidalClientId?: string;
	    tidalClientSecret?: string;
	    downloadFolder?: string;
	    downloadQuality?: string;
	    fileNameFormat?: string;
	    organizeFolders?: boolean;
	    embedCover: boolean;
	    concurrentDownloads?: number;
	    theme: string;
	    accentColor?: string;
	    soundEffects: boolean;
	    soundVolume: number;
	    embedLyrics: boolean;
	    preferSyncedLyrics: boolean;
	    tidalEnabled: boolean;
	    qobuzEnabled: boolean;
	    qobuzAppId?: string;
	    qobuzAppSecret?: string;
	    qobuzAuthToken?: string;
	    preferredSource?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tidalClientId = source["tidalClientId"];
	        this.tidalClientSecret = source["tidalClientSecret"];
	        this.downloadFolder = source["downloadFolder"];
	        this.downloadQuality = source["downloadQuality"];
	        this.fileNameFormat = source["fileNameFormat"];
	        this.organizeFolders = source["organizeFolders"];
	        this.embedCover = source["embedCover"];
	        this.concurrentDownloads = source["concurrentDownloads"];
	        this.theme = source["theme"];
	        this.accentColor = source["accentColor"];
	        this.soundEffects = source["soundEffects"];
	        this.soundVolume = source["soundVolume"];
	        this.embedLyrics = source["embedLyrics"];
	        this.preferSyncedLyrics = source["preferSyncedLyrics"];
	        this.tidalEnabled = source["tidalEnabled"];
	        this.qobuzEnabled = source["qobuzEnabled"];
	        this.qobuzAppId = source["qobuzAppId"];
	        this.qobuzAppSecret = source["qobuzAppSecret"];
	        this.qobuzAuthToken = source["qobuzAuthToken"];
	        this.preferredSource = source["preferredSource"];
	    }
	}
	export class ConversionFormat {
	    id: string;
	    name: string;
	    extension: string;
	    qualities: string[];
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversionFormat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.extension = source["extension"];
	        this.qualities = source["qualities"];
	        this.description = source["description"];
	    }
	}
	export class ConversionResult {
	    sourcePath: string;
	    outputPath: string;
	    success: boolean;
	    error?: string;
	    outputSize?: number;
	    sourceSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new ConversionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcePath = source["sourcePath"];
	        this.outputPath = source["outputPath"];
	        this.success = source["success"];
	        this.error = source["error"];
	        this.outputSize = source["outputSize"];
	        this.sourceSize = source["sourceSize"];
	    }
	}
	export class DownloadRecord {
	    id: number;
	    tidalContentId: string;
	    tidalContentName: string;
	    contentType: string;
	    // Go type: time
	    lastDownloadAt: any;
	    tracksTotal: number;
	    tracksDownloaded: number;
	    tracksFailed: number;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tidalContentId = source["tidalContentId"];
	        this.tidalContentName = source["tidalContentName"];
	        this.contentType = source["contentType"];
	        this.lastDownloadAt = this.convertValues(source["lastDownloadAt"], null);
	        this.tracksTotal = source["tracksTotal"];
	        this.tracksDownloaded = source["tracksDownloaded"];
	        this.tracksFailed = source["tracksFailed"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadResult {
	    trackId: number;
	    title: string;
	    artist: string;
	    album: string;
	    filePath: string;
	    fileSize: number;
	    quality: string;
	    coverUrl: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trackId = source["trackId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.filePath = source["filePath"];
	        this.fileSize = source["fileSize"];
	        this.quality = source["quality"];
	        this.coverUrl = source["coverUrl"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class DownloadedFileInfo {
	    path: string;
	    name: string;
	    size: number;
	    modTime: string;
	    title: string;
	    artist: string;
	    album: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadedFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	    }
	}
	export class FLACMetadata {
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    trackNumber: string;
	    date: string;
	    genre: string;
	    isrc: string;
	    comment: string;
	    size: number;
	    duration: number;
	    sampleRate: number;
	    bitDepth: number;
	    channels: number;
	    bitrate: number;
	    hasCover: boolean;
	    coverMime?: string;
	    coverSize?: number;
	    totalSamples: number;
	    lyrics?: string;
	    syncedLyrics?: string;
	    hasLyrics: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FLACMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.trackNumber = source["trackNumber"];
	        this.date = source["date"];
	        this.genre = source["genre"];
	        this.isrc = source["isrc"];
	        this.comment = source["comment"];
	        this.size = source["size"];
	        this.duration = source["duration"];
	        this.sampleRate = source["sampleRate"];
	        this.bitDepth = source["bitDepth"];
	        this.channels = source["channels"];
	        this.bitrate = source["bitrate"];
	        this.hasCover = source["hasCover"];
	        this.coverMime = source["coverMime"];
	        this.coverSize = source["coverSize"];
	        this.totalSamples = source["totalSamples"];
	        this.lyrics = source["lyrics"];
	        this.syncedLyrics = source["syncedLyrics"];
	        this.hasLyrics = source["hasLyrics"];
	    }
	}
	export class LogEntry {
	    timestamp: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class Lyrics {
	    plain: string;
	    synced: string;
	    source: string;
	    hasSynced: boolean;
	    trackName: string;
	    artistName: string;
	    albumName: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new Lyrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plain = source["plain"];
	        this.synced = source["synced"];
	        this.source = source["source"];
	        this.hasSynced = source["hasSynced"];
	        this.trackName = source["trackName"];
	        this.artistName = source["artistName"];
	        this.albumName = source["albumName"];
	        this.duration = source["duration"];
	    }
	}
	export class MatchFailure {
	    id: number;
	    tidalTrackId: string;
	    isrc: string;
	    title: string;
	    artist: string;
	    album: string;
	    reason: string;
	    attempts: number;
	    // Go type: time
	    lastAttemptAt: any;
	
	    static createFrom(source: any = {}) {
	        return new MatchFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tidalTrackId = source["tidalTrackId"];
	        this.isrc = source["isrc"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.reason = source["reason"];
	        this.attempts = source["attempts"];
	        this.lastAttemptAt = this.convertValues(source["lastAttemptAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SpotifyTrack {
	    id: string;
	    name: string;
	    artists: string;
	    album: string;
	    durationMs: number;
	    uri: string;
	    isrc?: string;
	
	    static createFrom(source: any = {}) {
	        return new SpotifyTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.artists = source["artists"];
	        this.album = source["album"];
	        this.durationMs = source["durationMs"];
	        this.uri = source["uri"];
	        this.isrc = source["isrc"];
	    }
	}
	export class TidalTrack {
	    id: number;
	    title: string;
	    artist: string;
	    artists: string;
	    album: string;
	    albumId: number;
	    isrc: string;
	    duration: number;
	    trackNumber: number;
	    coverUrl: string;
	    explicit: boolean;
	    tidalUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new TidalTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.artists = source["artists"];
	        this.album = source["album"];
	        this.albumId = source["albumId"];
	        this.isrc = source["isrc"];
	        this.duration = source["duration"];
	        this.trackNumber = source["trackNumber"];
	        this.coverUrl = source["coverUrl"];
	        this.explicit = source["explicit"];
	        this.tidalUrl = source["tidalUrl"];
	    }
	}
	export class MatchResult {
	    tidalTrack: TidalTrack;
	    spotifyTrack?: SpotifyTrack;
	    matched: boolean;
	    matchMethod: string;
	    confidence: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tidalTrack = this.convertValues(source["tidalTrack"], TidalTrack);
	        this.spotifyTrack = this.convertValues(source["spotifyTrack"], SpotifyTrack);
	        this.matched = source["matched"];
	        this.matchMethod = source["matchMethod"];
	        this.confidence = source["confidence"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RenamePreview {
	    oldPath: string;
	    oldName: string;
	    newName: string;
	    newPath: string;
	    hasError: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RenamePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldPath = source["oldPath"];
	        this.oldName = source["oldName"];
	        this.newName = source["newName"];
	        this.newPath = source["newPath"];
	        this.hasError = source["hasError"];
	        this.error = source["error"];
	    }
	}
	export class RenameResult {
	    oldPath: string;
	    newPath: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RenameResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldPath = source["oldPath"];
	        this.newPath = source["newPath"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class SourceTrack {
	    id: string;
	    title: string;
	    artist: string;
	    artists: string[];
	    album: string;
	    albumId: string;
	    isrc: string;
	    duration: number;
	    trackNumber: number;
	    totalTracks: number;
	    discNumber: number;
	    year: string;
	    genre: string;
	    coverUrl: string;
	    explicit: boolean;
	    sourceUrl: string;
	    source: string;
	    quality: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.artists = source["artists"];
	        this.album = source["album"];
	        this.albumId = source["albumId"];
	        this.isrc = source["isrc"];
	        this.duration = source["duration"];
	        this.trackNumber = source["trackNumber"];
	        this.totalTracks = source["totalTracks"];
	        this.discNumber = source["discNumber"];
	        this.year = source["year"];
	        this.genre = source["genre"];
	        this.coverUrl = source["coverUrl"];
	        this.explicit = source["explicit"];
	        this.sourceUrl = source["sourceUrl"];
	        this.source = source["source"];
	        this.quality = source["quality"];
	    }
	}
	export class SourceAlbum {
	    id: string;
	    title: string;
	    artist: string;
	    artists: string[];
	    year: string;
	    genre: string;
	    coverUrl: string;
	    trackCount: number;
	    tracks: SourceTrack[];
	    source: string;
	    sourceUrl: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceAlbum(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.artists = source["artists"];
	        this.year = source["year"];
	        this.genre = source["genre"];
	        this.coverUrl = source["coverUrl"];
	        this.trackCount = source["trackCount"];
	        this.tracks = this.convertValues(source["tracks"], SourceTrack);
	        this.source = source["source"];
	        this.sourceUrl = source["sourceUrl"];
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SourceInfo {
	    name: string;
	    displayName: string;
	    available: boolean;
	    urlPattern: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.available = source["available"];
	        this.urlPattern = source["urlPattern"];
	    }
	}
	export class SourcePlaylist {
	    id: string;
	    title: string;
	    description: string;
	    creator: string;
	    coverUrl: string;
	    trackCount: number;
	    tracks: SourceTrack[];
	    source: string;
	    sourceUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new SourcePlaylist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.creator = source["creator"];
	        this.coverUrl = source["coverUrl"];
	        this.trackCount = source["trackCount"];
	        this.tracks = this.convertValues(source["tracks"], SourceTrack);
	        this.source = source["source"];
	        this.sourceUrl = source["sourceUrl"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class TidalPlaylist {
	    uuid: string;
	    title: string;
	    description: string;
	    creator: string;
	    coverUrl: string;
	    numberOfTracks: number;
	    tracks: TidalTrack[];
	
	    static createFrom(source: any = {}) {
	        return new TidalPlaylist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.creator = source["creator"];
	        this.coverUrl = source["coverUrl"];
	        this.numberOfTracks = source["numberOfTracks"];
	        this.tracks = this.convertValues(source["tracks"], TidalTrack);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

