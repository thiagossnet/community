// Copyright 2018 Documize Inc. <legal@documize.com>. All rights reserved.
//
// This software (Documize Community Edition) is licensed under
// GNU AGPL v3 http://www.gnu.org/licenses/agpl-3.0.en.html
//
// You can operate outside the AGPL restrictions by purchasing
// Documize Enterprise Edition and obtaining a commercial license
// by contacting <sales@documize.com>.
//
// https://documize.com

package group

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/documize/community/core/env"
	"github.com/documize/community/core/request"
	"github.com/documize/community/core/response"
	"github.com/documize/community/core/uniqueid"
	"github.com/documize/community/domain"
	"github.com/documize/community/model/audit"
	"github.com/documize/community/model/group"
)

// Handler contains the runtime information such as logging and database.
type Handler struct {
	Runtime *env.Runtime
	Store   *domain.Store
}

// Add saves new user group.
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	method := "group.Add"
	ctx := domain.GetRequestContext(r)

	if !ctx.Administrator {
		response.WriteForbiddenError(w)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		response.WriteBadRequestError(w, method, "body")
		h.Runtime.Log.Error(method, err)
		return
	}

	var g group.Group
	err = json.Unmarshal(body, &g)
	if err != nil {
		response.WriteBadRequestError(w, method, "group")
		h.Runtime.Log.Error(method, err)
		return
	}

	g.RefID = uniqueid.Generate()
	g.OrgID = ctx.OrgID

	if len(g.Name) == 0 {
		response.WriteMissingDataError(w, method, "name")
		return
	}

	ctx.Transaction, err = h.Runtime.Db.Beginx()
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	err = h.Store.Group.Add(ctx, g)
	if err != nil {
		ctx.Transaction.Rollback()
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	ctx.Transaction.Commit()

	g, err = h.Store.Group.Get(ctx, g.RefID)
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	h.Store.Audit.Record(ctx, audit.EventTypeGroupAdd)

	response.WriteJSON(w, g)
}

// Groups returns all user groups for org.
func (h *Handler) Groups(w http.ResponseWriter, r *http.Request) {
	method := "group.Groups"
	ctx := domain.GetRequestContext(r)

	g, err := h.Store.Group.GetAll(ctx)
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	response.WriteJSON(w, g)
}

// Update saves group name and description changes.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	method := "group.Update"
	ctx := domain.GetRequestContext(r)

	if !ctx.Administrator {
		response.WriteForbiddenError(w)
		return
	}

	groupID := request.Param(r, "groupID")
	if len(groupID) == 0 {
		response.WriteMissingDataError(w, method, "groupID")
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		response.WriteBadRequestError(w, method, "body")
		h.Runtime.Log.Error(method, err)
		return
	}

	var g group.Group
	err = json.Unmarshal(body, &g)
	if err != nil {
		response.WriteBadRequestError(w, method, "group")
		h.Runtime.Log.Error(method, err)
		return
	}

	g.OrgID = ctx.OrgID
	g.RefID = groupID

	ctx.Transaction, err = h.Runtime.Db.Beginx()
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	err = h.Store.Group.Update(ctx, g)
	if err != nil {
		ctx.Transaction.Rollback()
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	ctx.Transaction.Commit()

	h.Store.Audit.Record(ctx, audit.EventTypeGroupUpdate)

	g, err = h.Store.Group.Get(ctx, g.RefID)
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	response.WriteJSON(w, g)
}

// Delete removes group and associated member records.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	method := "group.Delete"
	ctx := domain.GetRequestContext(r)

	if !ctx.Administrator {
		response.WriteForbiddenError(w)
		return
	}

	groupID := request.Param(r, "groupID")
	if len(groupID) == 0 {
		response.WriteMissingDataError(w, method, "groupID")
		return
	}

	g, err := h.Store.Group.Get(ctx, groupID)
	if err != nil {
		response.WriteServerError(w, method, err)
		return
	}

	ctx.Transaction, err = h.Runtime.Db.Beginx()
	if err != nil {
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	_, err = h.Store.Group.Delete(ctx, g.RefID)
	if err != nil {
		ctx.Transaction.Rollback()
		response.WriteServerError(w, method, err)
		h.Runtime.Log.Error(method, err)
		return
	}

	ctx.Transaction.Commit()

	h.Store.Audit.Record(ctx, audit.EventTypeGroupDelete)

	response.WriteEmpty(w)
}
