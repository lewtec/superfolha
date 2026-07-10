import type { TFunction } from "i18next";
import { ErrorCode } from "./errorCodes";

type GraphQLLikeError = {
  message?: string;
  extensions?: { code?: string; [key: string]: unknown };
};

/** Extract stable error code from Relay/GraphQL or REST JSON. */
export function extractErrorCode(err: unknown): string {
  if (!err) return ErrorCode.UNKNOWN;

  // REST body { code }
  if (typeof err === "object" && err !== null && "code" in err) {
    const c = (err as { code?: string }).code;
    if (typeof c === "string" && c) return c;
  }

  // Error with cause / nested
  if (err instanceof Error) {
    // Relay payload errors often on (err as any).source.errors
    const anyErr = err as Error & {
      source?: { errors?: GraphQLLikeError[] };
      errors?: GraphQLLikeError[];
    };
    const list =
      anyErr.source?.errors ||
      anyErr.errors ||
      (Array.isArray((err as { graphQLErrors?: GraphQLLikeError[] }).graphQLErrors)
        ? (err as { graphQLErrors: GraphQLLikeError[] }).graphQLErrors
        : null);
    if (list?.[0]?.extensions?.code) {
      return String(list[0].extensions.code);
    }
  }

  // Array of GraphQL errors
  if (Array.isArray(err) && err[0]?.extensions?.code) {
    return String(err[0].extensions.code);
  }

  // Single GraphQL error shape
  if (
    typeof err === "object" &&
    err !== null &&
    "extensions" in err &&
    (err as GraphQLLikeError).extensions?.code
  ) {
    return String((err as GraphQLLikeError).extensions!.code);
  }

  return ErrorCode.UNKNOWN;
}

export function translateError(t: TFunction, err: unknown): string {
  const code = extractErrorCode(err);
  const key = `errors:${code}`;
  const translated = t(key);
  if (translated === key || translated === code) {
    return t("errors:UNKNOWN");
  }
  return translated;
}

/** For Relay onCompleted errors array */
export function translateGraphQLErrors(
  t: TFunction,
  errors: ReadonlyArray<{ message: string; extensions?: { code?: string } }> | null | undefined,
): string {
  if (!errors?.length) return t("errors:UNKNOWN");
  return translateError(t, errors[0]);
}
