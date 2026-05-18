package biz

import utils_v1 "github.com/makesalekz/utils/api/utils/v1"

func replyPaginate(paginate *utils_v1.PaginateRequest, length int, total int32, fromId, toId *int64) *utils_v1.PaginateReply {
	paginateReply := utils_v1.PaginateReply{
		Total: &total,
	}

	// default descending order, updates / reverse pagination by FromId
	if paginate.GetToId() == 0 || paginate.GetFromId() != 0 || paginate.GetAroundId() != 0 {
		if fromId != nil {
			paginateReply.FromId = fromId
		}
	}
	// pagination by ToId
	if (paginate.GetFromId() == 0 || paginate.GetAroundId() != 0) && length == int(paginate.GetLimit()) {
		if toId != nil {
			paginateReply.ToId = toId
		}
	}

	return &paginateReply
}
