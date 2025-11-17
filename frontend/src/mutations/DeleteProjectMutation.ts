import { graphql } from "react-relay";

export default graphql`
  mutation DeleteProjectMutation($id: ID!) {
    deleteProject(id: $id)
  }
`;
