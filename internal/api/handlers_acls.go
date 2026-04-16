package api

import (
	"net/http"

	dbpkg "mediamtx-ui/internal/db"
)

type aclResponse struct {
	ID            int64  `json:"id"`
	SubjectType   string `json:"subjectType"`
	SubjectID     int64  `json:"subjectId"`
	SubjectName   string `json:"subjectName"`
	StreamPattern string `json:"streamPattern"`
	Action        string `json:"action"`
	CreatedAt     string `json:"createdAt"`
}

type createACLRequest struct {
	SubjectType   string `json:"subjectType"`
	SubjectID     int64  `json:"subjectId"`
	StreamPattern string `json:"streamPattern"`
	Action        string `json:"action"`
}

func (s *Server) handleListACLs(w http.ResponseWriter, r *http.Request) {
	acls, err := dbpkg.ListACLs(s.db)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]aclResponse, len(acls))
	for i, a := range acls {
		out[i] = toACLResponse(a)
	}
	jsonOK(w, out)
}

func (s *Server) handleCreateACL(w http.ResponseWriter, r *http.Request) {
	var req createACLRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.StreamPattern == "" || req.SubjectID == 0 {
		jsonErr(w, http.StatusBadRequest, "subjectId and streamPattern required")
		return
	}

	subType := dbpkg.SubjectType(req.SubjectType)
	if subType != dbpkg.SubjectUser && subType != dbpkg.SubjectGroup {
		jsonErr(w, http.StatusBadRequest, "subjectType must be 'user' or 'group'")
		return
	}

	action := dbpkg.ACLAction(req.Action)
	if action != dbpkg.ACLActionRead && action != dbpkg.ACLActionPublish && action != dbpkg.ACLActionBoth {
		jsonErr(w, http.StatusBadRequest, "action must be 'read', 'publish', or 'both'")
		return
	}

	acl, err := dbpkg.CreateACL(s.db, subType, req.SubjectID, req.StreamPattern, action)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonCreated(w, toACLResponse(acl))
}

func (s *Server) handleDeleteACL(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := dbpkg.DeleteACL(s.db, id); err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "acl not found")
		return
	} else if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	noContent(w)
}

func toACLResponse(a *dbpkg.ACL) aclResponse {
	return aclResponse{
		ID:            a.ID,
		SubjectType:   string(a.SubjectType),
		SubjectID:     a.SubjectID,
		SubjectName:   a.SubjectName,
		StreamPattern: a.StreamPattern,
		Action:        string(a.Action),
		CreatedAt:     a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
