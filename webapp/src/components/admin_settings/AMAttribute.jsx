import React, { useState} from 'react';
import PropTypes from 'prop-types';

const AMAttribute = (props) => {
    const initialSettings = props.attributes === undefined || Object.keys(props.attributes).length === 0 ? {
        alertmanagerurl: "",
        channel: "",
        team: "",
        token: "",
        titletemplate: "",
        messagetemplate: "",

    } : {
        alertmanagerurl: props.attributes.alertmanagerurl? props.attributes.alertmanagerurl: "",
        channel: props.attributes.channel? props.attributes.channel : "",
        team: props.attributes.team ? props.attributes.team: "",
        token: props.attributes.token? props.attributes.token: "",
        titletemplate: props.attributes.titletemplate || "",
        messagetemplate: props.attributes.messagetemplate || "",

    };

    const initErrors = {
        teamError: false,
        channelError: false,
        urlError: false
    };

    const [ settings, setSettings ] = useState(initialSettings);
    const [ hasError, setHasError ] = useState(initErrors);

    const handleTeamNameInput = (e) => {
        let newSettings = {...settings};

        if (!e.target.value || e.target.value.trim() === '') {
            setHasError({...hasError, teamError: true});
        } else {
            setHasError({...hasError, teamError: false});
        }

        newSettings = {...newSettings, team: e.target.value};

        setSettings(newSettings);
        props.onChange({id: props.id, attributes: newSettings});
    }

    const handleChannelNameInput = (e) => {
        let newSettings = {...settings};

        if (!e.target.value || e.target.value.trim() === '') {
            setHasError({...hasError, channelError: true});
        } else {
            setHasError({...hasError, channelError: false});
        }

        newSettings = {...newSettings, channel: e.target.value};

        setSettings(newSettings);
        props.onChange({id: props.id, attributes: newSettings});
    }

    const handleURLInput = (e) => {
        let newSettings = {...settings};

        if (!e.target.value || e.target.value.trim() === '') {
            setHasError({...hasError, urlError: true});
        } else {
            setHasError({...hasError, urlError: false});
        }

        newSettings = {...newSettings, alertmanagerurl: e.target.value};

        setSettings(newSettings);
        props.onChange({id: props.id, attributes: newSettings});
    }

    const handleDelete = () => {
        props.onDelete(props.id);
    }

    const handleTemplateInput = (e) => {
        const newSettings = {...settings, [e.target.name]: e.target.value};
        setSettings(newSettings);
        props.onChange({id: props.id, attributes: newSettings});
    }

    const regenerateToken = (e) => {
        e.preventDefault();

        // Generate a URL-safe token with 192 bits of browser-provided randomness.
        const token = Array.from(window.crypto.getRandomValues(new Uint8Array(24)), (byte) => byte.toString(16).padStart(2, '0')).join('');

        let newSettings = {...settings};
        newSettings = {...newSettings, token: token};

        setSettings(newSettings);
        props.onChange({id: props.id, attributes:newSettings});
    }

    const generateSimpleStringInputSetting = ( title, settingName, onChangeFunction, helpTextJSX) => {
        return (
            <div className="form-group" >
            <label className="control-label col-sm-4">
                {title}
            </label>
            <div className="col-sm-8">
                <input
                    id={`PluginSettings.Plugins.alertmanager.${settingName + "." + props.id}`}
                    className="form-control"
                    type="input"
                    onChange={onChangeFunction}
                    value={settings[settingName]}
                />
                <div className="help-text">
                    {helpTextJSX}
                </div>
            </div>
        </div>
        );
    }

    const generateGeneratedFieldSetting = ( title, settingName, regenerateFunction, regenerateText, helpTextJSX) => {
        return (<div className="form-group" >
        <label className="control-label col-sm-4">
            {title}
        </label>
        <div className="col-sm-8">
            <div
                id={`PluginSettings.Plugins.alertmanager.${settingName + "." + props.id}`}
                className="form-control disabled"
                >
                {settings[settingName] !== undefined && settings[settingName] !== ""? settings[settingName] : <span className="placeholder-text"></span>}
            </div>
            <div className="help-text">
                {helpTextJSX}
            </div>
            <div className="help-text">
                <button
                    type="button"
                    className="btn btn-default"
                    onClick={regenerateFunction}
                >
                    <span>{regenerateText}</span>
                </button>
            </div>
        </div>
    </div>);
    }

    const hasAnyError = () => {
        return Object.values(hasError).findIndex(item => item) !== -1;
    }

    return (
        <div id={`setting_${props.id}`} className={`alert-setting ${hasAnyError() ? 'alert-setting--with-error' : ''}`}>
            <div className='alert-setting__controls'>
                <div className='alert-setting__order-number'>{`#${props.id}`}</div>
                <div id={`delete_${props.id}`} className='delete-setting btn btn-default' onClick={handleDelete}>{` X `}</div>
            </div>
            { hasAnyError() && <div className='alert-setting__error-text'>{`Attribute cannot be empty.`}</div> }
            <div className='alert-setting__content'>
                <div>
                    { generateSimpleStringInputSetting(
                        "Team Name:",
                        "team",
                        handleTeamNameInput,
                        (<span>{"Team you want to send messages to. Use the team name such as 'my-team', instead of the display name."}</span>)
                        )
                    }

                    { generateSimpleStringInputSetting(
                        "Channel Name:",
                        "channel",
                        handleChannelNameInput,
                        (<span>{"Channel you want to send messages to. Use the channel name such as 'town-square', instead of the display name. If you specify a channel that does not exist, this plugin creates a new channel with that name."}</span>)
                        )
                    }

                    { generateGeneratedFieldSetting(
                        "Token:",
                        "token",
                        regenerateToken,
                        "Regenerate",
                        (<span>{"The token used to configure the webhook for AlertManager. The token is validates for each webhook request by the Mattermost server."}</span>)
                        )
                    }

                    { generateSimpleStringInputSetting(
                        "AlertManager URL:",
                        "alertmanagerurl",
                        handleURLInput,
                        (<span>{"The URL of your AlertManager instance, e.g. '"}<a href="http://alertmanager.example.com/" rel="noopener noreferrer" target="_blank">{"http://alertmanager.example.com/"}</a>{"'"}</span>)
                        )
                    }
                    <details className="alert-setting__templates">
                        <summary>{'Custom notification templates (optional)'}</summary>
                        <p className="help-text">
                            {'Leave both fields empty to keep the default format. Custom templates produce compact alerts and keep their status colors. Invalid templates use the default format.'}
                        </p>
                        {[
                            ['titletemplate', 'Title template:', 2, '{{ .Status | toUpper }}: {{ .Labels.alertname }}'],
                            ['messagetemplate', 'Message template:', 4, 'alertname={{ .Labels.alertname }}, instance={{ .Labels.instance }}, severity={{ .Labels.severity }}'],
                        ].map(([name, label, rows, placeholder]) => (
                            <div className="form-group" key={name}>
                                <label className="control-label col-sm-4" htmlFor={`PluginSettings.Plugins.alertmanager.${name}.${props.id}`}>
                                    {label}
                                </label>
                                <div className="col-sm-8">
                                    <textarea
                                        id={`PluginSettings.Plugins.alertmanager.${name}.${props.id}`}
                                        name={name}
                                        className="form-control alert-setting__template-input"
                                        rows={rows}
                                        maxLength={16384}
                                        placeholder={placeholder}
                                        value={settings[name]}
                                        onChange={handleTemplateInput}
                                    />
                                </div>
                            </div>
                        ))}
                        <p className="help-text">
                            {'Templates run once per alert. '}
                            <a href="https://github.com/cpanato/mattermost-plugin-alertmanager/blob/main/docs/templates.md" target="_blank" rel="noopener noreferrer">
                                {'See the available fields, examples and preview CLI.'}
                            </a>
                        </p>
                    </details>
                </div>
            </div>
        </div>
    );
}

AMAttribute.propTypes = {
    id: PropTypes.string.isRequired,
    orderNumber: PropTypes.number.isRequired,
    attributes: PropTypes.object,
    onDelete: PropTypes.func.isRequired,
    onChange: PropTypes.func.isRequired
}

export default AMAttribute;
