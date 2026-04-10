// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"
)

// Server implements ServerInterface by delegating to a raw UFM client.
type Server struct {
	ufm *ufmclient.Client
}

// NewServer creates a new Server backed by the given UFM client.
func NewServer(ufm *ufmclient.Client) *Server {
	return &Server{ufm: ufm}
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T { return &v }

// --------------------------------------------------------------------------
// Pagination helpers
// --------------------------------------------------------------------------

const defaultLimit = 100
const maxLimit = 1000

// resolveLimit clamps the user-supplied limit to [1, maxLimit].
func resolveLimit(l *int) int {
	if l == nil || *l <= 0 {
		return defaultLimit
	}
	if *l > maxLimit {
		return maxLimit
	}
	return *l
}

// decodeCursor decodes a base64-encoded cursor into an integer offset.
func decodeCursor(c *string) int {
	if c == nil || *c == "" {
		return 0
	}
	b, err := base64.StdEncoding.DecodeString(*c)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(b))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// encodeCursor returns a base64-encoded cursor for the given offset, or nil.
func encodeCursor(offset int) *string {
	s := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
	return &s
}

// paginate returns a slice of items and a PageInfo for cursor-based pagination.
func paginate[T any](items []T, offset, limit int) ([]T, PageInfo) {
	total := len(items)
	if offset >= total {
		return nil, PageInfo{Total: &total}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := items[offset:end]
	pi := PageInfo{Total: &total}
	if end < total {
		pi.NextCursor = encodeCursor(end)
	}
	return page, pi
}

// includesField returns true if the comma-separated include string contains field.
func includesField(include *string, field string) bool {
	if include == nil {
		return false
	}
	for _, f := range strings.Split(*include, ",") {
		if strings.TrimSpace(f) == field {
			return true
		}
	}
	return false
}

// ============================================================================
// Partitions (PKEY lifecycle)
// ============================================================================

func (s *Server) ListPartitions(ctx echo.Context, params ListPartitionsParams) error {
	guidsData := includesField(params.Include, "members")
	qosConf := includesField(params.Include, "qos")
	portInfo := includesField(params.Include, "port_info")
	sharpState := includesField(params.Include, "sharp_state")

	// When we need detail, use the detailed map endpoint.
	if guidsData || qosConf {
		detailed, err := s.ufm.ListPKeysDetailed(ctx.Request().Context())
		if err != nil {
			return handleUFMError(ctx, err)
		}
		var all []Partition
		for pkey, pk := range detailed {
			p := translatePartition(pkey, &pk, guidsData, qosConf)
			all = append(all, p)
		}
		offset := decodeCursor(params.Cursor)
		limit := resolveLimit(params.Limit)
		page, pi := paginate(all, offset, limit)
		return ctx.JSON(http.StatusOK, PartitionList{Items: &page, PageInfo: &pi})
	}

	pkeys, err := s.ufm.ListPKeys(ctx.Request().Context(), false, false, portInfo, sharpState, 0)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	var all []Partition
	for _, pk := range pkeys {
		all = append(all, Partition{Pkey: ptr(pk)})
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, PartitionList{Items: &page, PageInfo: &pi})
}

func (s *Server) CreatePartition(ctx echo.Context) error {
	var body PartitionCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}

	idx0 := body.Index0 != nil && *body.Index0
	ipoib := body.IpOverIb != nil && *body.IpOverIb

	if body.Members != nil && len(*body.Members) > 0 {
		guids := make([]string, len(*body.Members))
		memberships := make([]string, len(*body.Members))
		for i, m := range *body.Members {
			guids[i] = m.Guid
			if m.Membership != nil {
				memberships[i] = string(*m.Membership)
			} else {
				memberships[i] = "full"
			}
		}
		req := &ufmclient.PKeyAddGUIDsRequest{
			GUIDs:         guids,
			PKey:          body.Pkey,
			IPOverIB:      ipoib,
			Index0:        idx0,
			Memberships:   memberships,
		}
		if body.Name != nil {
			req.PartitionName = *body.Name
		}
		if err := s.ufm.AddGUIDsToPKey(ctx.Request().Context(), req); err != nil {
			return handleUFMError(ctx, err)
		}
	} else {
		req := &ufmclient.PKeyCreateRequest{
			PKey:     body.Pkey,
			Index0:   idx0,
			IPOverIB: ipoib,
		}
		if body.Name != nil {
			req.PartitionName = *body.Name
		}
		if body.Qos != nil {
			if body.Qos.MtuLimit != nil {
				req.MTULimit = ptr(int(*body.Qos.MtuLimit))
			}
			if body.Qos.ServiceLevel != nil {
				req.ServiceLevel = body.Qos.ServiceLevel
			}
			if body.Qos.RateLimit != nil {
				req.RateLimit = ptr(float64(*body.Qos.RateLimit))
			}
		}
		if err := s.ufm.CreateEmptyPKey(ctx.Request().Context(), req); err != nil {
			return handleUFMError(ctx, err)
		}
	}
	return ctx.JSON(http.StatusCreated, Partition{Pkey: &body.Pkey})
}

func (s *Server) DeletePartition(ctx echo.Context, pkey Pkey) error {
	if err := s.ufm.DeletePKey(ctx.Request().Context(), pkey); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) GetPartition(ctx echo.Context, pkey Pkey, params GetPartitionParams) error {
	guidsData := includesField(params.Include, "members")
	pk, err := s.ufm.GetPKey(ctx.Request().Context(), pkey, guidsData)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	p := translatePartition(pkey, pk, guidsData, false)
	return ctx.JSON(http.StatusOK, p)
}

func (s *Server) ReplacePartition(ctx echo.Context, pkey Pkey) error {
	var body PartitionReplace
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}

	guids := make([]string, len(body.Members))
	memberships := make([]string, len(body.Members))
	for i, m := range body.Members {
		guids[i] = m.Guid
		if m.Membership != nil {
			memberships[i] = string(*m.Membership)
		} else {
			memberships[i] = "full"
		}
	}

	idx0 := body.Index0 != nil && *body.Index0
	ipoib := body.IpOverIb != nil && *body.IpOverIb

	req := &ufmclient.PKeySetGUIDsRequest{
		GUIDs:       guids,
		PKey:        pkey,
		IPOverIB:    ipoib,
		Index0:      idx0,
		Memberships: memberships,
	}
	if body.Name != nil {
		req.PartitionName = *body.Name
	}
	if body.Qos != nil {
		if body.Qos.MtuLimit != nil {
			req.MTULimit = ptr(int(*body.Qos.MtuLimit))
		}
		if body.Qos.ServiceLevel != nil {
			req.ServiceLevel = body.Qos.ServiceLevel
		}
		if body.Qos.RateLimit != nil {
			req.RateLimit = ptr(float64(*body.Qos.RateLimit))
		}
	}

	if err := s.ufm.SetPKeyGUIDs(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ============================================================================
// Partition Members (GUID/host management)
// ============================================================================

func (s *Server) ListPartitionMembers(ctx echo.Context, pkey Pkey, params ListPartitionMembersParams) error {
	pk, err := s.ufm.GetPKey(ctx.Request().Context(), pkey, true)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	var members []PartitionMember
	for _, g := range pk.GUIDs {
		m := PartitionMember{
			Guid:   ptr(g.GUID),
			Index0: ptr(g.Index0),
		}
		if g.Membership != "" {
			ms := PartitionMemberMembership(g.Membership)
			m.Membership = &ms
		}
		members = append(members, m)
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(members, offset, limit)
	return ctx.JSON(http.StatusOK, PartitionMemberList{Items: &page, PageInfo: &pi})
}

func (s *Server) AddPartitionMembers(ctx echo.Context, pkey Pkey) error {
	var body PartitionMembersAdd
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}

	guids := make([]string, len(body.Members))
	memberships := make([]string, len(body.Members))
	for i, m := range body.Members {
		guids[i] = m.Guid
		if m.Membership != nil {
			memberships[i] = string(*m.Membership)
		} else {
			memberships[i] = "full"
		}
	}

	idx0 := body.Index0 != nil && *body.Index0
	ipoib := body.IpOverIb != nil && *body.IpOverIb

	req := &ufmclient.PKeyAddGUIDsRequest{
		GUIDs:       guids,
		PKey:        pkey,
		IPOverIB:    ipoib,
		Index0:      idx0,
		Memberships: memberships,
	}
	if err := s.ufm.AddGUIDsToPKey(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) RemovePartitionMembers(ctx echo.Context, pkey Pkey) error {
	var body RemovePartitionMembersJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.RemoveGUIDsFromPKey(ctx.Request().Context(), pkey, body.Guids); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) AddPartitionHosts(ctx echo.Context, pkey Pkey) error {
	var body PartitionHostsAdd
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}

	membership := "full"
	if body.Membership != nil {
		membership = string(*body.Membership)
	}
	idx0 := body.Index0 != nil && *body.Index0
	ipoib := body.IpOverIb != nil && *body.IpOverIb

	req := &ufmclient.PKeyAddHostsRequest{
		HostsNames: strings.Join(body.Hosts, ","),
		PKey:       pkey,
		IPOverIB:   ipoib,
		Index0:     idx0,
		Membership: membership,
	}
	job, err := s.ufm.AddHostsToPKey(ctx.Request().Context(), req)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusAccepted, Job{Id: ptr(job.ID)})
}

func (s *Server) RemovePartitionHosts(ctx echo.Context, pkey Pkey) error {
	var body RemovePartitionHostsJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.RemoveHostsFromPKey(ctx.Request().Context(), pkey, body.Hosts); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) GetPartitionLastUpdated(ctx echo.Context, pkey Pkey) error {
	result, err := s.ufm.GetPKeyLastUpdated(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"last_updated": result.LastUpdated,
	})
}

func (s *Server) UpdatePartitionQoS(ctx echo.Context, pkey Pkey) error {
	var body PartitionQoS
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.PKeyQoSRequest{PKey: pkey}
	if body.MtuLimit != nil {
		req.MTULimit = int(*body.MtuLimit)
	}
	if body.ServiceLevel != nil {
		req.ServiceLevel = *body.ServiceLevel
	}
	if body.RateLimit != nil {
		req.RateLimit = float64(*body.RateLimit)
	}
	if err := s.ufm.UpdatePKeyQoS(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) UpdatePartitionSHARP(ctx echo.Context, pkey Pkey) error {
	var body UpdatePartitionSHARPJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.SetPKeySHARP(ctx.Request().Context(), pkey, body.Enabled); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// translatePartition converts a ufmclient.PKey to the generated Partition type.
func translatePartition(pkey string, pk *ufmclient.PKey, includeMembers, includeQoS bool) Partition {
	p := Partition{
		Pkey:         ptr(pkey),
		Name:         ptr(pk.Partition),
		IpOverIb:     ptr(pk.IPOverIB),
		SharpEnabled: ptr(pk.SharpEnabled),
	}
	if includeMembers && len(pk.GUIDs) > 0 {
		members := make([]PartitionMember, len(pk.GUIDs))
		for i, g := range pk.GUIDs {
			ms := PartitionMemberMembership(g.Membership)
			members[i] = PartitionMember{
				Guid:       ptr(g.GUID),
				Index0:     ptr(g.Index0),
				Membership: &ms,
			}
		}
		p.Members = &members
	}
	if includeQoS {
		qos := PartitionQoS{}
		if pk.MTULimit != nil {
			ml := PartitionQoSMtuLimit(*pk.MTULimit)
			qos.MtuLimit = &ml
		}
		if pk.ServiceLevel != nil {
			qos.ServiceLevel = pk.ServiceLevel
		}
		if pk.RateLimit != nil {
			rl := float32(*pk.RateLimit)
			qos.RateLimit = &rl
		}
		p.Qos = &qos
	}
	return p
}

// ============================================================================
// Systems
// ============================================================================

func (s *Server) ListSystems(ctx echo.Context, params ListSystemsParams) error {
	opts := &ufmclient.ListSystemsOptions{}
	if params.Type != nil {
		opts.Type = string(*params.Type)
	}
	if params.Model != nil {
		opts.Model = *params.Model
	}
	if params.Role != nil {
		opts.Role = string(*params.Role)
	}
	if params.Ip != nil {
		opts.IP = *params.Ip
	}
	if params.Rack != nil {
		opts.Rack = *params.Rack
	}
	if params.ComputeStatus != nil {
		opts.Computes = string(*params.ComputeStatus)
	}
	if includesField(params.Include, "ports") {
		opts.Ports = true
	}
	if includesField(params.Include, "chassis") {
		opts.Chassis = true
	}
	if includesField(params.Include, "brief") {
		opts.Brief = true
	}

	systems, err := s.ufm.ListSystems(ctx.Request().Context(), opts)
	if err != nil {
		return handleUFMError(ctx, err)
	}

	all := make([]System, len(systems))
	for i, sys := range systems {
		all[i] = translateSystem(&sys)
	}

	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, SystemList{Items: &page, PageInfo: &pi})
}

func (s *Server) GetSystem(ctx echo.Context, systemId string, params GetSystemParams) error {
	sys, err := s.ufm.GetSystem(ctx.Request().Context(), systemId)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := translateSystem(sys)
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) UpdateSystem(ctx echo.Context, systemId string) error {
	var body SystemUpdate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}

	update := make(map[string]interface{})
	if body.Name != nil {
		update["system_name"] = *body.Name
	}
	if body.Ip != nil {
		update["ip"] = *body.Ip
	}

	if len(update) > 0 {
		if err := s.ufm.UpdateSystem(ctx.Request().Context(), systemId, update); err != nil {
			return handleUFMError(ctx, err)
		}
	}

	props := make(map[string]interface{})
	if body.Url != nil {
		props["url"] = *body.Url
	}
	if body.Script != nil {
		props["script"] = *body.Script
	}
	if len(props) > 0 {
		if err := s.ufm.UpdateSystemProperties(ctx.Request().Context(), systemId, props); err != nil {
			return handleUFMError(ctx, err)
		}
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ListSystemsPower(ctx echo.Context) error {
	powers, err := s.ufm.GetSystemsPower(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]SystemPower, len(powers))
	for i, p := range powers {
		result[i] = SystemPower{
			SystemId:   ptr(p.SystemID),
			PowerWatts: ptr(float32(p.PowerWatts)),
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

func translateSystem(sys *ufmclient.System) System {
	s := System{
		Guid: ptr(sys.SystemGUID),
	}
	if sys.SystemName != "" {
		s.Name = ptr(sys.SystemName)
	}
	if sys.SystemIP != "" {
		s.Ip = ptr(sys.SystemIP)
	}
	if sys.Type != "" {
		t := SystemType(sys.Type)
		s.Type = &t
	}
	if sys.Model != "" {
		s.Model = ptr(sys.Model)
	}
	if sys.Role != "" {
		r := SystemRole(sys.Role)
		s.Role = &r
	}
	if sys.Vendor != "" {
		s.Vendor = ptr(sys.Vendor)
	}
	if sys.FirmwareVersion != "" {
		s.FirmwareVersion = ptr(sys.FirmwareVersion)
	}
	if sys.Description != "" {
		s.Description = ptr(sys.Description)
	}
	return s
}

// ============================================================================
// Ports
// ============================================================================

func (s *Server) ListPorts(ctx echo.Context, params ListPortsParams) error {
	opts := &ufmclient.ListPortsOptions{}
	if params.System != nil {
		opts.System = *params.System
	}
	if params.SystemType != nil {
		opts.SystemType = *params.SystemType
	}
	if params.Active != nil && *params.Active {
		opts.Active = true
	}
	if params.External != nil && *params.External {
		opts.External = true
	}
	if params.HighBer != nil && *params.HighBer {
		opts.HighBER = true
	}
	if params.HighBerSeverity != nil {
		opts.HighBERSeverity = string(*params.HighBerSeverity)
	}
	if includesField(params.Include, "cable_info") {
		opts.CableInfo = true
	}

	ports, err := s.ufm.ListPorts(ctx.Request().Context(), opts)
	if err != nil {
		return handleUFMError(ctx, err)
	}

	all := make([]Port, len(ports))
	for i, p := range ports {
		all[i] = translatePort(&p)
	}

	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, PortList{Items: &page, PageInfo: &pi})
}

func (s *Server) GetPort(ctx echo.Context, portName string) error {
	ports, err := s.ufm.GetPorts(ctx.Request().Context(), []string{portName})
	if err != nil {
		return handleUFMError(ctx, err)
	}
	if len(ports) == 0 {
		return apiError(ctx, http.StatusNotFound, "not_found", fmt.Sprintf("port %s not found", portName))
	}
	p := translatePort(&ports[0])
	return ctx.JSON(http.StatusOK, p)
}

func (s *Server) DisablePort(ctx echo.Context, portName string) error {
	if err := s.ufm.DisablePort(ctx.Request().Context(), portName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func (s *Server) EnablePort(ctx echo.Context, portName string) error {
	if err := s.ufm.EnablePort(ctx.Request().Context(), portName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func (s *Server) ResetPort(ctx echo.Context, portName string) error {
	if err := s.ufm.ResetPort(ctx.Request().Context(), portName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func translatePort(p *ufmclient.Port) Port {
	port := Port{
		Name:               ptr(p.Name),
		Guid:               ptr(p.GUID),
		SystemName:         ptr(p.SystemName),
		SystemGuid:         ptr(p.SystemGUID),
		PhysicalPortNumber: ptr(p.PhysicalPortNumber),
		External:           ptr(p.External),
	}
	if p.LogicalPortState != "" {
		port.LogicalPortState = ptr(p.LogicalPortState)
	}
	if p.PhysicalPortState != "" {
		port.PhysicalPortState = ptr(p.PhysicalPortState)
	}
	if p.Speed != "" {
		port.Speed = ptr(p.Speed)
	}
	if p.Width != "" {
		port.Width = ptr(p.Width)
	}
	return port
}

// ============================================================================
// Links
// ============================================================================

func (s *Server) ListLinks(ctx echo.Context, params ListLinksParams) error {
	links, err := s.ufm.ListLinks(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	all := make([]Link, len(links))
	for i, l := range links {
		all[i] = Link{
			SourcePort:      ptr(l.SourcePort),
			DestinationPort: ptr(l.DestinationPort),
			Speed:           ptr(l.Speed),
			Width:           ptr(l.Width),
		}
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, LinkList{Items: &page, PageInfo: &pi})
}

func (s *Server) CollectLinkDump(ctx echo.Context) error {
	var body CollectLinkDumpJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.CollectSystemDump(ctx.Request().Context(), body.LinkIds); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

// ============================================================================
// Virtual Ports
// ============================================================================

func (s *Server) ListVirtualPorts(ctx echo.Context, params ListVirtualPortsParams) error {
	system := ""
	port := ""
	if params.System != nil {
		system = *params.System
	}
	if params.Port != nil {
		port = *params.Port
	}
	vports, err := s.ufm.ListVirtualPorts(ctx.Request().Context(), system, port)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	all := make([]VirtualPort, len(vports))
	for i, v := range vports {
		all[i] = VirtualPort{
			VirtualPortGuid:    ptr(v.VirtualPortGUID),
			VirtualPortState:   ptr(v.VirtualPortState),
			VirtualPortLid:     ptr(v.VirtualPortLID),
			SystemGuid:         ptr(v.SystemGUID),
			SystemName:         ptr(v.SystemName),
			SystemIp:           ptr(v.SystemIP),
			PortGuid:           ptr(v.PortGUID),
			PortName:           ptr(v.PortName),
			PhysicalPortNumber: ptr(v.PhysicalPortNumber),
		}
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, VirtualPortList{Items: &page, PageInfo: &pi})
}

// ============================================================================
// Jobs
// ============================================================================

func (s *Server) ListJobs(ctx echo.Context, params ListJobsParams) error {
	opts := &ufmclient.ListJobsOptions{}
	if params.Status != nil {
		opts.Status = string(*params.Status)
	}
	if params.Operation != nil {
		opts.Operation = *params.Operation
	}
	if params.ParentOnly != nil && *params.ParentOnly {
		opts.ParentOnly = true
	}
	if params.SystemIds != nil {
		opts.SystemIDs = *params.SystemIds
	}

	jobs, err := s.ufm.ListJobs(ctx.Request().Context(), opts)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	all := make([]Job, len(jobs))
	for i, j := range jobs {
		all[i] = translateJob(&j)
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, JobList{Items: &page, PageInfo: &pi})
}

func (s *Server) GetJob(ctx echo.Context, jobId string, params GetJobParams) error {
	advanced := includesField(params.Include, "sub_jobs") || includesField(params.Include, "full")
	j, err := s.ufm.GetJob(ctx.Request().Context(), jobId, advanced)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := translateJob(j)
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) DeleteJob(ctx echo.Context, jobId string) error {
	if err := s.ufm.DeleteJob(ctx.Request().Context(), jobId); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) AbortAllJobs(ctx echo.Context) error {
	if err := s.ufm.AbortAllJobs(ctx.Request().Context()); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) AbortJob(ctx echo.Context, jobId string) error {
	// UFM does not have per-job abort; abort all is the only available API.
	if err := s.ufm.AbortAllJobs(ctx.Request().Context()); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func translateJob(j *ufmclient.Job) Job {
	job := Job{
		Id: ptr(j.ID),
	}
	if j.Operation != "" {
		job.Operation = ptr(j.Operation)
	}
	if j.Status != "" {
		st := JobStatus(strings.ToLower(j.Status))
		job.Status = &st
	}
	return job
}

// ============================================================================
// Monitoring
// ============================================================================

func (s *Server) ListMonitoringAttributes(ctx echo.Context) error {
	result, err := s.ufm.GetMonitoringAttributes(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetCongestionMap(ctx echo.Context) error {
	result, err := s.ufm.GetCongestion(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetPerformanceCounters(ctx echo.Context) error {
	var body GetPerformanceCountersJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	result, err := s.ufm.GetPerformanceCounters(ctx.Request().Context(), body.Hostnames)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetPortGroupMetrics(ctx echo.Context) error {
	result, err := s.ufm.GetPortGroups(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) CreateMonitoringSession(ctx echo.Context) error {
	var body MonitoringSessionCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MonitoringSession{}
	if body.Attributes != nil {
		req.Attributes = *body.Attributes
	}
	if body.Interval != nil {
		req.Interval = *body.Interval
	}
	if body.Members != nil {
		req.Members = *body.Members
	}
	sess, err := s.ufm.CreateMonitoringSession(ctx.Request().Context(), req)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := MonitoringSession{
		Id:         ptr(sess.ID),
		Status:     ptr(sess.Status),
		Interval:   ptr(sess.Interval),
		Attributes: &sess.Attributes,
	}
	return ctx.JSON(http.StatusCreated, result)
}

func (s *Server) GetMonitoringSession(ctx echo.Context, sessionId string) error {
	sess, err := s.ufm.GetMonitoringSession(ctx.Request().Context(), sessionId)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := MonitoringSession{
		Id:         ptr(sess.ID),
		Status:     ptr(sess.Status),
		Interval:   ptr(sess.Interval),
		Attributes: &sess.Attributes,
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetMonitoringSessionData(ctx echo.Context, sessionId string, params GetMonitoringSessionDataParams) error {
	pkey := ""
	if params.Partition != nil {
		pkey = *params.Partition
	}
	result, err := s.ufm.GetMonitoringSessionData(ctx.Request().Context(), sessionId, pkey)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) DeleteMonitoringSession(ctx echo.Context, sessionId string) error {
	if err := s.ufm.DeleteMonitoringSession(ctx.Request().Context(), sessionId); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) CreateMonitoringSnapshot(ctx echo.Context) error {
	var body MonitoringSnapshotCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MonitoringSession{}
	if body.Attributes != nil {
		req.Attributes = *body.Attributes
	}
	if body.Members != nil {
		req.Members = *body.Members
	}
	result, err := s.ufm.CreateMonitoringSnapshot(ctx.Request().Context(), req)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) ListMonitoringTemplates(ctx echo.Context) error {
	tmpls, err := s.ufm.ListMonitoringTemplates(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]MonitoringTemplate, len(tmpls))
	for i, t := range tmpls {
		result[i] = MonitoringTemplate{
			Name:       ptr(t.Name),
			Interval:   ptr(t.Interval),
			Attributes: &t.Attributes,
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetMonitoringTemplate(ctx echo.Context, templateName string) error {
	t, err := s.ufm.GetMonitoringTemplate(ctx.Request().Context(), templateName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, MonitoringTemplate{
		Name:       ptr(t.Name),
		Interval:   ptr(t.Interval),
		Attributes: &t.Attributes,
	})
}

func (s *Server) CreateMonitoringTemplate(ctx echo.Context) error {
	var body MonitoringTemplate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MonitoringTemplate{}
	if body.Name != nil {
		req.Name = *body.Name
	}
	if body.Interval != nil {
		req.Interval = *body.Interval
	}
	if body.Attributes != nil {
		req.Attributes = *body.Attributes
	}
	if err := s.ufm.CreateMonitoringTemplate(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusCreated)
}

func (s *Server) UpdateMonitoringTemplate(ctx echo.Context, templateName string) error {
	var body MonitoringTemplate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MonitoringTemplate{Name: templateName}
	if body.Interval != nil {
		req.Interval = *body.Interval
	}
	if body.Attributes != nil {
		req.Attributes = *body.Attributes
	}
	if err := s.ufm.UpdateMonitoringTemplate(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteMonitoringTemplate(ctx echo.Context, templateName string) error {
	if err := s.ufm.DeleteMonitoringTemplate(ctx.Request().Context(), templateName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ============================================================================
// Telemetry
// ============================================================================

func (s *Server) GetHistoryTelemetry(ctx echo.Context, params GetHistoryTelemetryParams) error {
	result, err := s.ufm.GetHistoryTelemetry(ctx.Request().Context(),
		params.MembersType, params.Attributes, params.Members,
		params.StartTime, params.EndTime)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetTopTelemetry(ctx echo.Context, params GetTopTelemetryParams) error {
	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	attrs := ""
	if params.Attributes != nil {
		attrs = *params.Attributes
	}
	result, err := s.ufm.GetTopTelemetry(ctx.Request().Context(),
		params.MembersType, params.PickBy, limit, attrs)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

// ============================================================================
// Topology (Environments, Servers, Computes, Networks)
// ============================================================================

func (s *Server) ListEnvironments(ctx echo.Context, params ListEnvironmentsParams) error {
	envs, err := s.ufm.ListEnvironments(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	all := make([]Environment, len(envs))
	for i, e := range envs {
		all[i] = Environment{Name: ptr(e.Name), Description: ptr(e.Description)}
	}
	offset := decodeCursor(params.Cursor)
	limit := resolveLimit(params.Limit)
	page, pi := paginate(all, offset, limit)
	return ctx.JSON(http.StatusOK, EnvironmentList{Items: &page, PageInfo: &pi})
}

func (s *Server) CreateEnvironment(ctx echo.Context) error {
	var body EnvironmentCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	env := &ufmclient.Environment{Name: body.Name}
	if body.Description != nil {
		env.Description = *body.Description
	}
	if err := s.ufm.CreateEnvironment(ctx.Request().Context(), env); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, Environment{Name: ptr(body.Name), Description: body.Description})
}

func (s *Server) GetEnvironment(ctx echo.Context, envName string) error {
	env, err := s.ufm.GetEnvironment(ctx.Request().Context(), envName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, Environment{Name: ptr(env.Name), Description: ptr(env.Description)})
}

func (s *Server) UpdateEnvironment(ctx echo.Context, envName string) error {
	var body EnvironmentCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	env := &ufmclient.Environment{Name: body.Name}
	if body.Description != nil {
		env.Description = *body.Description
	}
	if err := s.ufm.UpdateEnvironment(ctx.Request().Context(), envName, env); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteEnvironment(ctx echo.Context, envName string) error {
	if err := s.ufm.DeleteEnvironment(ctx.Request().Context(), envName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ListLogicalServers(ctx echo.Context, envName string) error {
	servers, err := s.ufm.ListLogicalServers(ctx.Request().Context(), envName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]LogicalServer, len(servers))
	for i, srv := range servers {
		result[i] = LogicalServer{Name: ptr(srv.Name), Description: ptr(srv.Description)}
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) CreateLogicalServer(ctx echo.Context, envName string) error {
	var body LogicalServerCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	srv := &ufmclient.LogicalServer{Name: body.Name}
	if body.Description != nil {
		srv.Description = *body.Description
	}
	if err := s.ufm.CreateLogicalServer(ctx.Request().Context(), envName, srv); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, LogicalServer{Name: ptr(body.Name), Description: body.Description})
}

func (s *Server) GetLogicalServer(ctx echo.Context, envName string, serverName string) error {
	srv, err := s.ufm.GetLogicalServer(ctx.Request().Context(), envName, serverName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, LogicalServer{Name: ptr(srv.Name), Description: ptr(srv.Description)})
}

func (s *Server) DeleteLogicalServer(ctx echo.Context, envName string, serverName string) error {
	if err := s.ufm.DeleteLogicalServer(ctx.Request().Context(), envName, serverName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ListComputes(ctx echo.Context, envName string, serverName string) error {
	computes, err := s.ufm.ListComputes(ctx.Request().Context(), envName, serverName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]Compute, len(computes))
	for i, c := range computes {
		result[i] = Compute{Name: ptr(c.Name)}
		if c.Status != "" {
			st := ComputeStatus(c.Status)
			result[i].Status = &st
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) AssignComputes(ctx echo.Context, envName string, serverName string) error {
	var body AssignComputesJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := make(map[string]interface{})
	if body.Mode != nil {
		req["mode"] = string(*body.Mode)
	}
	if body.Computes != nil {
		req["computes"] = *body.Computes
	}
	if err := s.ufm.AllocateComputes(ctx.Request().Context(), envName, serverName, req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) FreeComputes(ctx echo.Context, envName string, serverName string) error {
	if err := s.ufm.FreeComputes(ctx.Request().Context(), envName, serverName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// Networks (global and local)

func (s *Server) ListGlobalNetworks(ctx echo.Context) error {
	nets, err := s.ufm.ListGlobalNetworks(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]Network, len(nets))
	for i, n := range nets {
		result[i] = translateNetwork(&n)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) CreateGlobalNetwork(ctx echo.Context) error {
	var body NetworkCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	net := &ufmclient.Network{Name: body.Name}
	if body.Pkey != nil {
		net.PKey = *body.Pkey
	}
	if body.IpOverIb != nil {
		net.IPOverIB = *body.IpOverIb
	}
	if err := s.ufm.CreateGlobalNetwork(ctx.Request().Context(), net); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, Network{Name: ptr(body.Name), Pkey: body.Pkey, IpOverIb: body.IpOverIb})
}

func (s *Server) GetGlobalNetwork(ctx echo.Context, networkName string) error {
	net, err := s.ufm.GetGlobalNetwork(ctx.Request().Context(), networkName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, translateNetwork(net))
}

func (s *Server) UpdateGlobalNetwork(ctx echo.Context, networkName string) error {
	var body NetworkCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	net := &ufmclient.Network{Name: body.Name}
	if body.Pkey != nil {
		net.PKey = *body.Pkey
	}
	if body.IpOverIb != nil {
		net.IPOverIB = *body.IpOverIb
	}
	if err := s.ufm.UpdateGlobalNetwork(ctx.Request().Context(), net); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteGlobalNetwork(ctx echo.Context, networkName string) error {
	if err := s.ufm.DeleteGlobalNetwork(ctx.Request().Context(), networkName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ListLocalNetworks(ctx echo.Context, envName string) error {
	nets, err := s.ufm.ListLocalNetworks(ctx.Request().Context(), envName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]Network, len(nets))
	for i, n := range nets {
		result[i] = translateNetwork(&n)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) CreateLocalNetwork(ctx echo.Context, envName string) error {
	var body NetworkCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	net := &ufmclient.Network{Name: body.Name}
	if body.Pkey != nil {
		net.PKey = *body.Pkey
	}
	if body.IpOverIb != nil {
		net.IPOverIB = *body.IpOverIb
	}
	if err := s.ufm.CreateLocalNetwork(ctx.Request().Context(), envName, net); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, Network{Name: ptr(body.Name), Pkey: body.Pkey, IpOverIb: body.IpOverIb})
}

func translateNetwork(n *ufmclient.Network) Network {
	net := Network{Name: ptr(n.Name)}
	if n.PKey != "" {
		net.Pkey = ptr(n.PKey)
	}
	net.IpOverIb = ptr(n.IPOverIB)
	return net
}

// ============================================================================
// Mirroring
// ============================================================================

func (s *Server) CreateMirroringTemplate(ctx echo.Context) error {
	var body MirroringTemplate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MirroringTemplate{}
	if body.SystemId != nil {
		req.SystemID = *body.SystemId
	}
	if body.TargetPort != nil {
		req.TargetPort = *body.TargetPort
	}
	if body.PacketSize != nil {
		req.PacketSize = *body.PacketSize
	}
	if body.ServiceLevel != nil {
		req.ServiceLevel = *body.ServiceLevel
	}
	if err := s.ufm.CreateMirroringTemplate(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusCreated)
}

func (s *Server) GetMirroringTemplate(ctx echo.Context, systemId string) error {
	t, err := s.ufm.GetMirroringTemplate(ctx.Request().Context(), systemId)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, MirroringTemplate{
		SystemId:     ptr(t.SystemID),
		TargetPort:   ptr(t.TargetPort),
		PacketSize:   ptr(t.PacketSize),
		ServiceLevel: ptr(t.ServiceLevel),
	})
}

func (s *Server) UpdateMirroringTemplate(ctx echo.Context, systemId string) error {
	var body MirroringTemplate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MirroringTemplate{SystemID: systemId}
	if body.TargetPort != nil {
		req.TargetPort = *body.TargetPort
	}
	if body.PacketSize != nil {
		req.PacketSize = *body.PacketSize
	}
	if body.ServiceLevel != nil {
		req.ServiceLevel = *body.ServiceLevel
	}
	if err := s.ufm.UpdateMirroringTemplate(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteMirroringTemplate(ctx echo.Context, systemId string) error {
	if err := s.ufm.DeleteMirroringTemplate(ctx.Request().Context(), systemId); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ExecuteMirroringAction(ctx echo.Context) error {
	var body MirroringAction
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.MirroringAction{
		PortID: body.PortId,
		Action: string(body.Action),
		RX:     body.Rx,
		TX:     body.Tx,
	}
	if err := s.ufm.ExecuteMirroringAction(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

// ============================================================================
// Templates (Provisioning)
// ============================================================================

func (s *Server) ListTemplates(ctx echo.Context, params ListTemplatesParams) error {
	tags := ""
	profile := ""
	systemType := ""
	if params.Tags != nil {
		tags = *params.Tags
	}
	if params.Profile != nil {
		profile = *params.Profile
	}
	if params.SystemType != nil {
		systemType = *params.SystemType
	}
	tmpls, err := s.ufm.ListTemplates(ctx.Request().Context(), tags, profile, systemType)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := make([]ProvisioningTemplate, len(tmpls))
	for i, t := range tmpls {
		result[i] = translateProvisioningTemplate(&t)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetTemplate(ctx echo.Context, templateName string) error {
	t, err := s.ufm.GetTemplate(ctx.Request().Context(), templateName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, translateProvisioningTemplate(t))
}

func (s *Server) CreateTemplate(ctx echo.Context) error {
	var body ProvisioningTemplateCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	content := make([]interface{}, len(body.Content))
	for i, c := range body.Content {
		content[i] = c
	}
	req := &ufmclient.ProvisioningTemplate{
		Title:      body.Title,
		SystemType: body.SystemType,
		Content:    content,
	}
	if body.Description != nil {
		req.Description = *body.Description
	}
	if err := s.ufm.CreateTemplate(ctx.Request().Context(), req); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusCreated)
}

func (s *Server) DeleteTemplate(ctx echo.Context, templateName string) error {
	if err := s.ufm.DeleteTemplate(ctx.Request().Context(), templateName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) RefreshTemplates(ctx echo.Context) error {
	if err := s.ufm.RefreshTemplates(ctx.Request().Context()); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ExecuteTemplate(ctx echo.Context, templateName string) error {
	var body ExecuteTemplateJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.ExecuteProvisioningTemplate(ctx.Request().Context(), templateName, body.SystemIds); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func translateProvisioningTemplate(t *ufmclient.ProvisioningTemplate) ProvisioningTemplate {
	pt := ProvisioningTemplate{
		Title: ptr(t.Title),
	}
	if t.Description != "" {
		pt.Description = ptr(t.Description)
	}
	if t.SystemType != "" {
		pt.SystemType = ptr(t.SystemType)
	}
	if t.Owner != "" {
		pt.Owner = ptr(t.Owner)
	}
	if len(t.Tags) > 0 {
		pt.Tags = &t.Tags
	}
	if len(t.Content) > 0 {
		content := make([]map[string]interface{}, len(t.Content))
		for i, c := range t.Content {
			if m, ok := c.(map[string]interface{}); ok {
				content[i] = m
			}
		}
		pt.Content = &content
	}
	return pt
}

// ============================================================================
// Fabric Health (validations, reports)
// ============================================================================

func (s *Server) ListFabricValidationTests(ctx echo.Context) error {
	tests, err := s.ufm.ListFabricValidationTests(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, tests)
}

func (s *Server) RunFabricValidationTest(ctx echo.Context, testName string) error {
	job, err := s.ufm.RunFabricValidationTest(ctx.Request().Context(), testName)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusAccepted, Job{Id: ptr(job.ID)})
}

func (s *Server) CreateReport(ctx echo.Context) error {
	var body ReportCreate
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	req := &ufmclient.ReportRequest{
		Type: string(body.Type),
	}
	if body.Parameters != nil {
		req.Parameters = *body.Parameters
	}
	job, err := s.ufm.CreateReport(ctx.Request().Context(), req)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusAccepted, Job{Id: ptr(job.ID)})
}

func (s *Server) GetReport(ctx echo.Context, reportId string) error {
	result, err := s.ufm.GetReport(ctx.Request().Context(), reportId)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetLatestReport(ctx echo.Context, reportType string) error {
	result, err := s.ufm.GetLatestReport(ctx.Request().Context(), reportType)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) DeleteReport(ctx echo.Context, reportId string) error {
	if err := s.ufm.DeleteReport(ctx.Request().Context(), reportId); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ============================================================================
// Discovery
// ============================================================================

func (s *Server) GetInventorySummary(ctx echo.Context, params GetInventorySummaryParams) error {
	showPorts := params.IncludePorts != nil && *params.IncludePorts
	result, err := s.ufm.GetInventory(ctx.Request().Context(), showPorts)
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetInventoryCount(ctx echo.Context) error {
	result, err := s.ufm.GetInventoryCount(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) RefreshFabricDiscovery(ctx echo.Context) error {
	if err := s.ufm.RefreshFabricDiscovery(ctx.Request().Context()); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

// ============================================================================
// Config
// ============================================================================

func (s *Server) GetConfig(ctx echo.Context) error {
	cfg, err := s.ufm.GetConfig(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := UFMConfig{
		SiteName:               ptr(cfg.SiteName),
		HaMode:                 ptr(cfg.HAMode),
		IsLocalUser:            ptr(cfg.IsLocalUser),
		DefaultSessionInterval: ptr(cfg.DefaultSessionInterval),
	}
	if len(cfg.DisabledFeatures) > 0 {
		result.DisabledFeatures = &cfg.DisabledFeatures
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) UpdateConfig(ctx echo.Context) error {
	var body UpdateConfigJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := s.ufm.UpdateConfig(ctx.Request().Context(), body); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ConfigureDumpStorage(ctx echo.Context) error {
	var body ConfigureDumpStorageJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	path := ""
	if body.StoragePath != nil {
		path = *body.StoragePath
	}
	if err := s.ufm.ConfigureDumpStorage(ctx.Request().Context(), path); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ============================================================================
// Cable Images
// ============================================================================

func (s *Server) ListCableImages(ctx echo.Context) error {
	images, err := s.ufm.ListCableImages(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, images)
}

func (s *Server) UploadCableImage(ctx echo.Context) error {
	// UFM cable image upload uses multipart form data. The raw client does not
	// expose this endpoint directly, so we proxy the multipart request as-is.
	file, err := ctx.FormFile("file")
	if err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", "missing file field")
	}
	src, err := file.Open()
	if err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	defer src.Close()

	if err := s.ufm.UploadCableImage(ctx.Request().Context(), file.Filename, src); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusCreated)
}

func (s *Server) DeleteCableImage(ctx echo.Context, imageName string) error {
	if err := s.ufm.DeleteCableImage(ctx.Request().Context(), imageName); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ============================================================================
// Actions (reboot, firmware, software, health, dump)
// ============================================================================

func (s *Server) RebootSystem(ctx echo.Context, systemId string) error {
	var body RebootSystemJSONRequestBody
	// Body is optional for reboot.
	_ = ctx.Bind(&body)

	if body.InBand != nil && *body.InBand {
		if err := s.ufm.InBandRebootSystem(ctx.Request().Context(), systemId); err != nil {
			return handleUFMError(ctx, err)
		}
	} else {
		if err := s.ufm.RebootSystem(ctx.Request().Context(), systemId); err != nil {
			return handleUFMError(ctx, err)
		}
	}
	return ctx.NoContent(http.StatusAccepted)
}

func (s *Server) UpgradeSystemFirmware(ctx echo.Context, systemId string) error {
	var body UpgradeSystemFirmwareJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	image := ""
	if body.Image != nil {
		image = *body.Image
	}
	inBand := body.InBand != nil && *body.InBand
	if err := s.ufm.UpgradeFirmware(ctx.Request().Context(), systemId, image, inBand); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func (s *Server) UpgradeSystemSoftware(ctx echo.Context, systemId string) error {
	var body UpgradeSystemSoftwareJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	image := ""
	if body.Image != nil {
		image = *body.Image
	}
	if err := s.ufm.UpgradeSoftware(ctx.Request().Context(), systemId, image); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

func (s *Server) SetSystemHealth(ctx echo.Context, systemId string) error {
	var body SetSystemHealthJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return apiError(ctx, http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.Healthy {
		if err := s.ufm.RestoreHealthy(ctx.Request().Context(), systemId); err != nil {
			return handleUFMError(ctx, err)
		}
	} else {
		policy := ""
		if body.IsolationPolicy != nil {
			policy = *body.IsolationPolicy
		}
		if err := s.ufm.MarkUnhealthy(ctx.Request().Context(), systemId, policy); err != nil {
			return handleUFMError(ctx, err)
		}
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) CollectSystemDump(ctx echo.Context, systemId string) error {
	if err := s.ufm.CollectSystemDump(ctx.Request().Context(), []string{systemId}); err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.NoContent(http.StatusAccepted)
}

// ============================================================================
// Tokens / Version
// ============================================================================

func (s *Server) CreateToken(ctx echo.Context) error {
	tok, err := s.ufm.CreateToken(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, map[string]interface{}{
		"token":      tok.Token,
		"expires_at": tok.ExpiresAt,
	})
}

func (s *Server) GetVersion(ctx echo.Context) error {
	v, err := s.ufm.GetVersion(ctx.Request().Context())
	if err != nil {
		return handleUFMError(ctx, err)
	}
	result := UFMVersion{
		UfmReleaseVersion: ptr(v.UFMReleaseVersion),
		OpensmVersion:     ptr(v.OpenSMVersion),
		SharpVersion:      ptr(v.SHARPVersion),
		IbdiagnetVersion:  ptr(v.IBDiagnetVersion),
		TelemetryVersion:  ptr(v.TelemetryVersion),
		MftVersion:        ptr(v.MFTVersion),
		WebuiVersion:      ptr(v.WebUIVersion),
	}
	if len(v.Plugins) > 0 {
		result.Plugins = &v.Plugins
	}
	return ctx.JSON(http.StatusOK, result)
}

// Compile-time interface compliance check.
var _ ServerInterface = (*Server)(nil)
