"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { errorMessage } from "@/lib/api";

type Snapshot<T> = {
  key: string;
  data: T | null;
  error: string | null;
};

/**
 * Small data hook for browser screens. The `key` string decides when to refetch.
 *
 * Loading is derived from the request key rather than set synchronously inside
 * an effect, which keeps renders predictable and avoids stale responses
 * overwriting newer ones.
 */
export function useApi<T>(loader: () => Promise<T>, key: string) {
  const loaderRef = useRef(loader);
  const [snapshot, setSnapshot] = useState<Snapshot<T>>({
    key: "",
    data: null,
    error: null,
  });
  const [nonce, setNonce] = useState(0);
  const requestKey = `${key}:${nonce}`;

  // Keep the latest loader without touching refs during render.
  useEffect(() => {
    loaderRef.current = loader;
  });

  useEffect(() => {
    let active = true;

    loaderRef
      .current()
      .then((data) => {
        if (active) setSnapshot({ key: requestKey, data, error: null });
      })
      .catch((error) => {
        if (active) {
          setSnapshot({ key: requestKey, data: null, error: errorMessage(error) });
        }
      });

    return () => {
      active = false;
    };
  }, [requestKey]);

  const current = snapshot.key === requestKey;
  const loading = !current;
  const data = snapshot.data;
  const error = current ? snapshot.error : null;

  const reload = useCallback(() => setNonce((n) => n + 1), []);

  return { data, loading, error, reload };
}

/** Tracks a one-shot async action (submit, approve, void, ...). */
export function useAction() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const run = useCallback(
    async <T>(action: () => Promise<T>, successMessage?: string) => {
      setPending(true);
      setError(null);
      setSuccess(null);
      try {
        const result = await action();
        if (successMessage) setSuccess(successMessage);
        return result;
      } catch (err) {
        setError(errorMessage(err));
        return null;
      } finally {
        setPending(false);
      }
    },
    [],
  );

  return { pending, error, success, run };
}
