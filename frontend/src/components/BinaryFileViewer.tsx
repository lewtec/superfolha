// frontend/src/components/BinaryFileViewer.tsx
import React from 'react';

interface BinaryFileViewerProps {
  fileName: string;
  fileContent: string; // Base64 encoded content
}

export default function BinaryFileViewer({ fileName, fileContent }: BinaryFileViewerProps) {
  const handleDownload = () => {
    try {
      // Decode base64 content
      const byteCharacters = atob(fileContent);
      const byteNumbers = new Array(byteCharacters.length);
      for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
      }
      const byteArray = new Uint8Array(byteNumbers);

      // Determine MIME type based on file extension (simple heuristic)
      const extension = fileName.split('.').pop()?.toLowerCase();
      let mimeType = 'application/octet-stream'; // Default binary type

      if (extension === 'pdf') mimeType = 'application/pdf';
      else if (extension === 'png') mimeType = 'image/png';
      else if (extension === 'jpg' || extension === 'jpeg') mimeType = 'image/jpeg';
      else if (extension === 'gif') mimeType = 'image/gif';
      else if (extension === 'zip') mimeType = 'application/zip';
      // Add more as needed

      const blob = new Blob([byteArray], { type: mimeType });
      const url = URL.createObjectURL(blob);

      const a = document.createElement('a');
      a.href = url;
      a.download = fileName;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Error downloading file:", error);
      alert("Failed to download file.");
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-full bg-base-100 text-base-content p-4">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-24 w-24 text-warning mb-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
        />
      </svg>
      <p className="text-lg font-semibold mb-2">Cannot display binary file: {fileName}</p>
      <p className="text-sm text-base-content/70 mb-4">
        This file appears to be a binary format and cannot be rendered directly in the editor.
      </p>
      <button className="btn btn-primary" onClick={handleDownload}>
        Download {fileName}
      </button>
    </div>
  );
}
