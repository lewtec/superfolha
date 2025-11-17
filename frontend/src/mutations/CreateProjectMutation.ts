import { graphql } from "react-relay";

export default graphql`
  mutation CreateProjectMutation($name: String!) {
    createProject(name: $name) {
      id
      name
      createdAt
      updatedAt
    }
  }
`;
