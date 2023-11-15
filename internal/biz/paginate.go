package biz

import utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"

func replyPaginate(paginate *utils_v1.PaginateRequest, length int, total int32, fromId, toId *int64) *utils_v1.PaginateReply {
	paginateReply := utils_v1.PaginateReply{
		Total: &total,
	}

	// default descending order, updates / reverse pagination by FromId
	if paginate.ToId == 0 || paginate.FromId != 0 || paginate.AroundId != 0 {
		if fromId != nil {
			paginateReply.FromId = fromId
		}
	}
	// pagination by ToId
	if (paginate.FromId == 0 || paginate.AroundId != 0) && length == int(paginate.Limit) {
		if toId != nil {
			paginateReply.ToId = toId
		}
	}

	return &paginateReply
}
