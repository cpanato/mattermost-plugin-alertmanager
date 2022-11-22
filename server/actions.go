package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/cpanato/mattermost-plugin-alertmanager/server/alertmanager"
)

func (p *Plugin) handleExpireAction(w http.ResponseWriter, r *http.Request, alertConfig alertConfig) {
	p.API.LogInfo("Received expire silence action")

	var action *Action
	_ = json.NewDecoder(r.Body).Decode(&action)

	if action == nil {
		encodeEphermalMessage(w, "We could not decode the action")
		return
	}

	if action.Context.SilenceID == "" {
		encodeEphermalMessage(w, "Silence ID cannot be empty")
		return
	}

	silenceDeletedMsg := fmt.Sprintf("Silence %s expired.", action.Context.SilenceID)

	err := alertmanager.ExpireSilence(action.Context.SilenceID, alertConfig.AlertManagerURL)
	if err != nil {
		msg := fmt.Sprintf("failed to expire the silence: %v", err)
		encodeEphermalMessage(w, msg)
	}

	updatePost := &model.Post{}

	attachments := []*model.SlackAttachment{}
	actionPost, errPost := p.API.GetPost(action.PostID)
	if errPost != nil {
		p.API.LogError("AlerManager Update Post Error", "err=", errPost.Error())
	} else {
		for _, attachment := range actionPost.Attachments() {
			if attachment.Actions == nil {
				attachments = append(attachments, attachment)
				continue
			}
			for _, actionItem := range attachment.Actions {
				if actionItem.Integration.Context["silence_id"] == action.Context.SilenceID {
					updateAttachment := attachment
					updateAttachment.Actions = nil
					updateAttachment.Color = colorExpired
					var silenceMsg string
					userName, errUser := p.API.GetUser(action.UserID)
					if errUser != nil {
						silenceMsg = "Silence expired"
					} else {
						silenceMsg = fmt.Sprintf("Silence expired by %s", userName.Username)
					}

					field := &model.SlackAttachmentField{
						Title: "Expired by",
						Value: silenceMsg,
						Short: false,
					}
					updateAttachment.Fields = append(updateAttachment.Fields, field)
					attachments = append(attachments, updateAttachment)
				} else {
					attachments = append(attachments, attachment)
				}
			}
		}
		retainedProps := []string{"override_username", "override_icon_url"}
		updatePost.AddProp("from_webhook", "true")

		for _, prop := range retainedProps {
			if value, ok := actionPost.Props[prop]; ok {
				updatePost.AddProp(prop, value)
			}
		}

		model.ParseSlackAttachment(updatePost, attachments)
		updatePost.Id = actionPost.Id
		updatePost.ChannelId = actionPost.ChannelId
		updatePost.UserId = actionPost.UserId
		if _, err := p.API.UpdatePost(updatePost); err != nil {
			encodeEphermalMessage(w, silenceDeletedMsg)
			return
		}
	}

	encodeEphermalMessage(w, silenceDeletedMsg)
}

func encodeEphermalMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]interface{}{
		"ephemeral_text": message,
	}

	_ = json.NewEncoder(w).Encode(payload)
}
