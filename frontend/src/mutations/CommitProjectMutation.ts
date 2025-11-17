import { graphql } from "react-relay";

export default graphql`
  mutation CommitProjectMutation($projectId: ID!, $message: String!) {
    commit(projectId: $projectId, message: $message) {
      hash
      message
      author
      date
    }
  }
`;
