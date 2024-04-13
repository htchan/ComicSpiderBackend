--
-- PostgreSQL database dump
--

-- Dumped from database version 14.2 (Debian 14.2-1.pgdg110+1)
-- Dumped by pg_dump version 14.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: user_websites; Type: TABLE; Schema: public; Owner: test
--

CREATE TABLE public.user_websites (
    website_uuid character varying(64),
    user_uuid character varying(64),
    access_time timestamp without time zone,
    group_name text
);


ALTER TABLE public.user_websites OWNER TO test;

--
-- Name: website_settings; Type: TABLE; Schema: public; Owner: test
--

CREATE TABLE public.website_settings (
    domain character varying(255),
    focus_index_from integer,
    focus_index_to integer,
    title_goquery_selector text,
    date_goquery_selector text
);


ALTER TABLE public.website_settings OWNER TO test;

--
-- Name: websites; Type: TABLE; Schema: public; Owner: test
--

CREATE TABLE public.websites (
    uuid character varying(64),
    url text,
    title text,
    content text,
    update_time timestamp without time zone
);


ALTER TABLE public.websites OWNER TO test;

--
-- Name: user_websites__user_and_uuid; Type: INDEX; Schema: public; Owner: test
--

CREATE UNIQUE INDEX user_websites__user_and_uuid ON public.user_websites USING btree (user_uuid, website_uuid);


--
-- Name: website_settings__domain; Type: INDEX; Schema: public; Owner: test
--

CREATE UNIQUE INDEX website_settings__domain ON public.website_settings USING btree (domain);


--
-- Name: websites__url; Type: INDEX; Schema: public; Owner: test
--

CREATE UNIQUE INDEX websites__url ON public.websites USING btree (url);


--
-- Name: websites__uuid; Type: INDEX; Schema: public; Owner: test
--

CREATE UNIQUE INDEX websites__uuid ON public.websites USING btree (uuid);


--
-- PostgreSQL database dump complete
--

