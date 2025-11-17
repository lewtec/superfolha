import { graphql } from "react-relay";

export default graphql`
  query ProjectsQuery {
    projects {
      id
      name
      createdAt
      updatedAt
    }
  }
`;
