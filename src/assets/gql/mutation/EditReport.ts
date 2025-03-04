import { Report } from "@structures/Report";
import gql from "graphql-tag";

export const EditReport = gql`
	mutation EditReport($id: String!, $data: EditReportInput!) {
		editReport(report_id: $id, data: $data) {
			id
			priority
			status
			assignees {
				id
				username
				avatar_url
				tag_color
			}
			subject
			body
			notes
		}
	}
`;

export interface EditReport {
	editReport: Report;
}

export namespace EditReport {
	export interface Variables {
		id: string;
		data: {
			status: Report.Status;
			assignee: string;
		};
	}
}
