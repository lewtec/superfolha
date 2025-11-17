import { graphql } from "react-relay";

export default graphql`
  query GetFileContentQuery($id: ID!, $path: String!) {
    project(id: $id) {
      file(path: $path) {
        content
      }
    }
  }
`;
