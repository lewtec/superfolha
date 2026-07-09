import {
  Environment,
  Network,
  RecordSource,
  Store,
  FetchFunction,
} from "relay-runtime";

const fetchQuery: FetchFunction = async (operation, variables) => {
  // Auth is cookie-based (HttpOnly authToken). Same-origin fetch sends cookies
  // by default; credentials: "same-origin" makes that explicit.
  const response = await fetch("/api/graphql", {
    method: "POST",
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      query: operation.text,
      variables,
    }),
  });

  return response.json();
};

const environment = new Environment({
  network: Network.create(fetchQuery),
  store: new Store(new RecordSource()),
});

export default environment;
