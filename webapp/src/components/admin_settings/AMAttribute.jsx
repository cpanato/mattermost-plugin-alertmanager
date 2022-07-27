import React, { useState, useEffect} from 'react';
import crypto from 'crypto';

const AMAttribute = (props) => {
    const initialSettings = props.attributes === undefined || Object.keys(props.attributes).length === 0 ? {
        id: props.id,
        teamName: "",
        channelName: "",
        token: "",
        url: "",
        error: "",
    } : {
        id: props.id,
        teamName: props.attributes.teamName ? props.attributes.teamName: "",
        channelName: props.attributes.channelName? props.attributes.channelName : "",
        token: props.attributes.token? props.attributes.token: "",
        url: props.attributes.url? props.attributes.url: "",
        error: "",
    };

    const [settings, setSettings] = useState(initialSettings);
    const [ hasError, setHasError ] = useState(false);

    /*useEffect(() => {

    });*/

    const handleTeamNameInput = (e) => {
        let newSettings = {...settings};
        setHasError(false);

        if (!e.target.value || e.target.value.trim() === '') {
            newSettings = {...newSettings, error: 'Attribute name cannot be empty.'};
            setHasError(true);
        }

        newSettings = {...newSettings, teamName: e.target.value};

        setSettings(newSettings);
        props.onChange({id: newSettings.id, attributes: newSettings});
    }

    const handleChannelNameInput = (e) => {
        let newSettings = {...settings};
        setHasError(false);

        if (!e.target.value || e.target.value.trim() === '') {
            newSettings = {...newSettings, error: 'Attribute name cannot be empty.'}
            setHasError(true);
        }

        newSettings = {...newSettings, channelName: e.target.value};

        setSettings(newSettings);
        props.onChange({id: newSettings.id, attributes: newSettings});
    }

    const handleURLInput = (e) => {
        let newSettings = {...settings};
        setHasError(false);

        if (!e.target.value || e.target.value.trim() === '') {
            newSettings = {...newSettings, error: 'Attribute name cannot be empty.'}
            setHasError(true);
        }

        newSettings = {...newSettings, url: e.target.value};

        setSettings(newSettings);
        props.onChange({id: newSettings.id, attributes: newSettings});
    }

    const handleDelete = (e) => {
        props.onDelete(props.id);
    }

    const regenerateToken = (e) => {
        e.preventDefault();

        const token =  crypto.randomBytes(256).toString('base64').substring(0, 32);
        //const token = generateRandomToken(32);

        let newSettings = {...settings};
        newSettings = {...newSettings, token: token};

        setSettings(newSettings);
        props.onChange({id: settings.id, attributes:newSettings});
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

    return (
        <div id={`setting_${props.id}`} className={`alert-setting ${hasError ? 'alert-setting--with-error' : ''}`}>
            <div className='alert-setting__controls'>
                <div className='alert-setting__id'>{`#${props.id}`}</div>
                <div id={`delete_${props.id}`} className='delete-setting btn btn-default' onClick={handleDelete}>{` X `}</div>
            </div>
            { hasError && <div className='alert-setting__error-text'>{`${settings.error}`}</div> }
            <div className='alert-setting__content'>
                <div>
                    { generateSimpleStringInputSetting(
                        "Team Name:",
                        "teamName",
                        handleTeamNameInput,
                        (<span>{"Team you want to send messages to. Use the team name such as \'my-team\', instead of the display name."}</span>)
                        )
                    }

                    { generateSimpleStringInputSetting(
                        "Channel Name:",
                        "channelName",
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
                        "url",
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
    id: PropTypes.number.isRequired,
    attributes: PropTypes.object,
    onChange: PropTypes.func.isRequired
}

export default AMAttribute;

function generateRandomToken(tokenLength){
    let token = "";

    let availableSymbols = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    for (let i = 0; i < tokenLength; i++) {
        token += availableSymbols.charAt(Math.floor(Math.random() * availableSymbols.length));
    }
    return token;
}