package api

import (
	"net/http"

	dbpkg "mediamtx-ui/internal/db"
)

type groupResponse struct {
	ID        int64          `json:"id"`
	Name      string         `json:"name"`
	Members   []userResponse `json:"members,omitempty"`
	CreatedAt string         `json:"createdAt"`
}

type createGroupRequest struct {
	Name string `json:"name"`
}

type updateGroupRequest struct {
	Name string `json:"name"`
}

type addMemberRequest struct {
	UserID int64 `json:"userId"`
}

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := dbpkg.ListGroups(s.db)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]groupResponse, len(groups))
	for i, g := range groups {
		out[i] = toGroupResponse(g)
	}
	jsonOK(w, out)
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		jsonErr(w, http.StatusBadRequest, "name required")
		return
	}

	g, err := dbpkg.CreateGroup(s.db, req.Name)
	if err == dbpkg.ErrConflict {
		jsonErr(w, http.StatusConflict, "group already exists")
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonCreated(w, toGroupResponse(g))
}

func (s *Server) handleRenameGroup(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateGroupRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		jsonErr(w, http.StatusBadRequest, "name required")
		return
	}
	g, err := dbpkg.RenameGroup(s.db, id, req.Name)
	if err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "group not found")
		return
	}
	if err == dbpkg.ErrConflict {
		jsonErr(w, http.StatusConflict, "group name already exists")
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, toGroupResponse(g))
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := dbpkg.DeleteGroup(s.db, id); err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "group not found")
		return
	} else if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	noContent(w)
}

func (s *Server) handleListGroupMembers(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	members, err := dbpkg.GetGroupMembers(s.db, id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]userResponse, len(members))
	for i, u := range members {
		out[i] = toUserResponse(u)
	}
	jsonOK(w, out)
}

func (s *Server) handleAddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid group id")
		return
	}
	var req addMemberRequest
	if err := decodeJSON(r, &req); err != nil || req.UserID == 0 {
		jsonErr(w, http.StatusBadRequest, "userId required")
		return
	}
	if err := dbpkg.AddGroupMember(s.db, groupID, req.UserID); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	noContent(w)
}

func (s *Server) handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid group id")
		return
	}
	userID, err := pathParamID(r, "userId")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := dbpkg.RemoveGroupMember(s.db, groupID, userID); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	noContent(w)
}

func toGroupResponse(g *dbpkg.Group) groupResponse {
	return groupResponse{
		ID:        g.ID,
		Name:      g.Name,
		CreatedAt: g.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// handleListGroupsWithMembers is called from the chi route with /{id} having members query param
func (s *Server) handleGetGroupWithMembers(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	g, err := dbpkg.GetGroupByID(s.db, id)
	if err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "group not found")
		return
	}
	members, err := dbpkg.GetGroupMembers(s.db, id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	gr := toGroupResponse(g)
	gr.Members = make([]userResponse, len(members))
	for i, u := range members {
		gr.Members[i] = toUserResponse(u)
	}
	jsonOK(w, gr)
}
