// frontend/src/utils/fileUtils.ts

const BINARY_EXTENSIONS = new Set([
  ".png",
  ".jpg",
  ".jpeg",
  ".gif",
  ".bmp",
  ".webp",
  ".ico", // Images
  ".pdf", // Documents
  ".zip",
  ".tar",
  ".gz",
  ".rar",
  ".7z", // Archives
  ".exe",
  ".dll",
  ".bin",
  ".out", // Executables/Binaries
  ".mp3",
  ".wav",
  ".ogg",
  ".flac", // Audio
  ".mp4",
  ".avi",
  ".mkv",
  ".mov", // Video
  ".woff",
  ".woff2",
  ".ttf",
  ".otf", // Fonts
  ".sqlite",
  ".db", // Databases
]);

export function isBinaryContent(content: string, filename: string): boolean {
  const lowerFilename = filename.toLowerCase();
  const extension = lowerFilename.substring(lowerFilename.lastIndexOf("."));

  // 1. Check by extension
  if (BINARY_EXTENSIONS.has(extension)) {
    return true;
  }

  // 2. Heuristic check for null bytes or non-printable characters
  // This is a heuristic and might not be 100% accurate.
  // We'll check the first N characters for null bytes or a high density of non-printable ASCII.
  const sampleSize = Math.min(content.length, 1024); // Check first 1KB
  let nonPrintableCount = 0;

  for (let i = 0; i < sampleSize; i++) {
    const charCode = content.charCodeAt(i);
    // Null byte
    if (charCode === 0) {
      return true;
    }
    // Non-printable ASCII characters (excluding common whitespace like tab, newline, carriage return)
    if (charCode < 32 && charCode !== 9 && charCode !== 10 && charCode !== 13) {
      nonPrintableCount++;
    }
  }

  // If more than 10% of the sample are non-printable (excluding common whitespace), consider it binary
  if (sampleSize > 0 && nonPrintableCount / sampleSize > 0.1) {
    return true;
  }

  return false;
}
