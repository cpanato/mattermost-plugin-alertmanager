import React, { useState, useEffect} from 'react';
import crypto from 'crypto';

const AMAttribute = (props) => {
    const initialSettings = props.attributes === undefined || Object.keys(props.attributes).length === 0 ? {
        alertmanagerurl: "",
        channel: "",
        team: "",
        token: "",

    } : {
        alertmanagerurl: props.attributes.alertmanagerurl? props.attributes.alertmanagerurl: "",
        channel: props.attributes.channel? props.attributes.channel : "",
        team: props.attributes.team ? props.attributes.team: "",
        token: props.attributes.token? props.attributes.token: "",

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

    const handleDelete = (e) => {
        props.onDelete(props.id);
    }

    const regenerateToken = (e) => {
        e.preventDefault();

        // Generate a 32 byte tokes. It must not include '*' and '/'.
        // Copied from https://github.com/mattermost/mattermost-webapp/blob/33661c60bd05d708bcf85a49dad4d9fb3a39a75b/components/admin_console/generated_setting.tsx#L41
        const token = crypto.randomBytes(256).toString('base64').substring(0, 32).replaceAll('+', '-').replaceAll('/', '_');

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
                    id={`PluginSettings.Plugins.alertmanager.${settingName + "." + settings.id}`}
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
                id={`PluginSettings.Plugins.alertmanager.${settingName + "." + settings.id}`}
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
                        (<span>{"Team you want to send messages to. Use the team name such as \'my-team\', instead of the display name."}</span>)
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
                        (<span>{"The URL of your AlertManager instance, e.g. \'"}<a href="http://alertmanager.example.com/" rel="noopener noreferrer" target="_blank">{"http://alertmanager.example.com/"}</a>{"\'"}</span>)
                        )
                    }
                </div>
            </div>
        </div>
    );
}

AMAttribute.propTypes = {
    id: PropTypes.string.isRequired,
    orderNumber: PropTypes.number.isRequired,
    attributes: PropTypes.object,
    onChange: PropTypes.func.isRequired
}

export default AMAttribute;
