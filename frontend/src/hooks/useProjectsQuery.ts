import { useLazyLoadQuery } from "react-relay";
import type { ProjectsQuery as ProjectsQueryType } from "../queries/__generated__/ProjectsQuery.graphql";
import ProjectsQueryGraphql from "../queries/ProjectsQuery";

interface UseProjectsQueryConfig {
  fetchPolicy?:
    | "store-and-network"
    | "network-only"
    | "store-only"
    | "store-or-network";
}

export function useProjectsQuery(config?: UseProjectsQueryConfig) {
  const data = useLazyLoadQuery<ProjectsQueryType>(
    ProjectsQueryGraphql,
    {},
    { fetchPolicy: config?.fetchPolicy || "store-and-network" },
  );

  return data;
}
