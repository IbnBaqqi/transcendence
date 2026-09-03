import { isApiError } from "../api/client";
import i18next from "../i18n";

export interface UploadFailure {
  message: string;
  /** Whether sending the same file again could plausibly work. */
  retryable: boolean;
}

export function describeUploadError(err: unknown, fileName: string): UploadFailure {
  if (!isApiError(err)) return { message: i18next.t("dropzone.uploadFailed"), retryable: false };

  switch (err.status) {
    // The client already passed its own 5 MiB check, so a 413 means the server's
    // limit is lower - naming a size here would name the wrong one.
    case 413:
      return { message: i18next.t("validation.uploadTooLarge"), retryable: false };
    // The browser trusts the extension; the server sniffs the bytes.
    case 415:
      return {
        message: i18next.t("validation.unsupportedImageType", { name: fileName }),
        retryable: false,
      };
    // status 0 is "never reached the server", already localized by client.ts.
    default:
      return { message: err.message, retryable: err.status === 0 || err.status >= 500 };
  }
}
